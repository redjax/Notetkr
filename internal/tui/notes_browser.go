package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type FilterMode int

const (
	FilterNone FilterMode = iota
	FilterSearch
	FilterTag
)

type directoryNode struct {
	name        string
	fullPath    string
	children    []*directoryNode
	isExpanded  bool
	hasChildren bool
}

type NotesBrowserModel struct {
	notesService       *services.NotesService
	notes              []services.Note
	filteredNotes      []services.Note
	directories        []string // Directories in current path
	currentPath        string   // Current navigation path relative to notes root
	allTags            []string
	templates          []services.Note
	cursor             int
	width              int
	height             int
	err                error
	searchInput        textinput.Model
	filterMode         FilterMode
	showingTags        bool
	showingTemplates   bool
	tagCursor          int
	templateCursor     int
	confirmDelete      bool
	deleteTarget       string
	deleteTargetIdx    int
	previewService     *services.PreviewService
	showingNewMenu     bool
	newMenuCursor      int
	creatingCategory   bool
	categoryInput      textinput.Model
	movingNote         bool
	moveTargetIdx      int
	moveInput          textinput.Model
	moveDirTree        []*directoryNode // Flattened view of directory tree for move UI
	moveCursor         int
	moveCreatingNewDir bool
}

var (
	notesBrowserTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(1, 0)

	noteItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	noteSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	noteTagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Italic(true)

	searchBarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170")).
			Padding(0, 1)

	tagListStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2)
)

func NewNotesBrowser(notesService *services.NotesService, width, height int) NotesBrowserModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search notes..."
	searchInput.CharLimit = 100

	categoryInput := textinput.New()
	categoryInput.Placeholder = "Enter category path (e.g., work/projects)..."
	categoryInput.CharLimit = 200
	categoryInput.Width = 50

	moveInput := textinput.New()
	moveInput.Placeholder = "Enter destination path (e.g., work/projects)..."
	moveInput.CharLimit = 200
	moveInput.Width = 50

	m := NotesBrowserModel{
		notesService:     notesService,
		searchInput:      searchInput,
		categoryInput:    categoryInput,
		moveInput:        moveInput,
		filterMode:       FilterNone,
		showingTags:      false,
		showingTemplates: false,
		width:            width,
		height:           height,
		previewService:   services.NewPreviewService(),
	}

	// Initialize default templates
	m.notesService.InitializeDefaultTemplates()

	m.loadNotes()
	m.loadTemplates()
	return m
}

func (m *NotesBrowserModel) loadNotes() {
	notes, directories, err := m.notesService.ListNotesInPath(m.currentPath)
	if err != nil {
		m.err = err
		return
	}

	m.notes = notes
	m.directories = directories
	m.filteredNotes = notes
	m.cursor = 0

	// Load all tags
	tags, err := m.notesService.GetAllTags()
	if err == nil {
		sort.Strings(tags)
		m.allTags = tags
	}
}

func (m *NotesBrowserModel) buildMoveDirectoryTree() {
	// Get all directories
	allDirs, err := m.notesService.GetAllDirectories()
	if err != nil {
		m.err = err
		return
	}

	// Build a tree structure
	root := &directoryNode{
		name:        ".",
		fullPath:    "",
		children:    []*directoryNode{},
		isExpanded:  true,
		hasChildren: false,
	}

	nodeMap := make(map[string]*directoryNode)
	nodeMap[""] = root

	// Create nodes for all directories
	for _, dirPath := range allDirs {
		parts := strings.Split(filepath.ToSlash(dirPath), "/")
		currentPath := ""

		for i, part := range parts {
			if i > 0 {
				currentPath = filepath.Join(currentPath, parts[i-1])
			}
			fullPath := filepath.Join(currentPath, part)

			if _, exists := nodeMap[fullPath]; !exists {
				node := &directoryNode{
					name:        part,
					fullPath:    fullPath,
					children:    []*directoryNode{},
					isExpanded:  false,
					hasChildren: false,
				}
				nodeMap[fullPath] = node

				// Add to parent
				parent := nodeMap[currentPath]
				parent.children = append(parent.children, node)
				parent.hasChildren = true
			}
			currentPath = fullPath
		}
	}

	// Flatten the tree for display
	m.moveDirTree = m.flattenDirectoryTree(root)
}

func (m *NotesBrowserModel) flattenDirectoryTree(node *directoryNode) []*directoryNode {
	result := []*directoryNode{}

	if node.fullPath != "" { // Don't include root
		result = append(result, node)
	}

	if node.isExpanded {
		for _, child := range node.children {
			result = append(result, m.flattenDirectoryTree(child)...)
		}
	}

	return result
}

func (m *NotesBrowserModel) toggleDirectoryExpansion() {
	if m.moveCursor < len(m.moveDirTree) {
		node := m.moveDirTree[m.moveCursor]
		if node.hasChildren {
			node.isExpanded = !node.isExpanded
			// Rebuild flattened tree
			root := m.findRootNode()
			m.moveDirTree = m.flattenDirectoryTree(root)
		}
	}
}

func (m *NotesBrowserModel) findRootNode() *directoryNode {
	// Reconstruct the tree from flattened list
	root := &directoryNode{
		name:        ".",
		fullPath:    "",
		children:    []*directoryNode{},
		isExpanded:  true,
		hasChildren: len(m.moveDirTree) > 0,
	}

	nodeMap := make(map[string]*directoryNode)
	nodeMap[""] = root

	for _, node := range m.moveDirTree {
		nodeMap[node.fullPath] = node

		// Find parent
		parentPath := filepath.Dir(node.fullPath)
		if parentPath == "." {
			parentPath = ""
		}

		if parent, exists := nodeMap[parentPath]; exists {
			// Check if already in children
			found := false
			for _, child := range parent.children {
				if child.fullPath == node.fullPath {
					found = true
					break
				}
			}
			if !found {
				parent.children = append(parent.children, node)
			}
		}
	}

	return root
}

func (m *NotesBrowserModel) loadTemplates() {
	templates, err := m.notesService.ListTemplates()
	if err != nil {
		m.err = err
		return
	}
	m.templates = templates
	m.templateCursor = 0
}

func (m NotesBrowserModel) Init() tea.Cmd {
	return nil
}

func (m NotesBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle delete confirmation
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				if m.deleteTargetIdx >= 0 && m.deleteTargetIdx < len(m.filteredNotes) {
					note := m.filteredNotes[m.deleteTargetIdx]
					err := m.notesService.DeleteNote(note.FilePath)
					if err != nil {
						m.err = err
					} else {
						m.loadNotes()
						if m.cursor >= len(m.filteredNotes) && m.cursor > 0 {
							m.cursor--
						}
					}
				}
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteTargetIdx = -1
				return m, nil

			case "n", "N", "esc":
				// Cancel delete
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteTargetIdx = -1
				return m, nil
			}
			return m, nil
		}

		// Handle tag selection mode
		if m.showingTags {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.showingTags = false
				return m, nil

			case "up", "k":
				if m.tagCursor > 0 {
					m.tagCursor--
				}
				return m, nil

			case "down", "j":
				if m.tagCursor < len(m.allTags)-1 {
					m.tagCursor++
				}
				return m, nil

			case "enter", "l":
				if len(m.allTags) > 0 && m.tagCursor < len(m.allTags) {
					selectedTag := m.allTags[m.tagCursor]
					notes, err := m.notesService.FilterByTag(selectedTag)
					if err == nil {
						m.filteredNotes = notes
						m.cursor = 0
						m.filterMode = FilterTag
					}
					m.showingTags = false
				}
				return m, nil
			}
			return m, nil
		}

		// Handle category input
		if m.creatingCategory {
			switch msg.String() {
			case "esc":
				m.creatingCategory = false
				m.categoryInput.Blur()
				m.categoryInput.SetValue("")
				return m, nil

			case "enter":
				categoryPath := strings.TrimSpace(m.categoryInput.Value())
				if categoryPath != "" {
					// Combine current path with new category path
					fullCategoryPath := categoryPath
					if m.currentPath != "" {
						fullCategoryPath = filepath.Join(m.currentPath, categoryPath)
					}
					// Create the category directory
					if err := m.notesService.CreateCategory(fullCategoryPath); err != nil {
						m.err = err
					} else {
						m.loadNotes()
					}
				}
				m.creatingCategory = false
				m.categoryInput.Blur()
				m.categoryInput.SetValue("")
				return m, nil

			default:
				var cmd tea.Cmd
				m.categoryInput, cmd = m.categoryInput.Update(msg)
				return m, cmd
			}
		}

		// Handle move note directory selection or new directory input
		if m.movingNote {
			// If creating new directory, handle text input
			if m.moveCreatingNewDir {
				switch msg.String() {
				case "esc":
					m.moveCreatingNewDir = false
					m.moveInput.Blur()
					m.moveInput.SetValue("")
					return m, nil

				case "enter":
					destPath := strings.TrimSpace(m.moveInput.Value())
					if destPath != "" && m.moveTargetIdx < len(m.filteredNotes) {
						note := m.filteredNotes[m.moveTargetIdx]
						// Move the note
						if err := m.notesService.MoveNote(note.FilePath, destPath); err != nil {
							m.err = err
						} else {
							m.loadNotes()
						}
					}
					m.movingNote = false
					m.moveCreatingNewDir = false
					m.moveInput.Blur()
					m.moveInput.SetValue("")
					return m, nil

				default:
					var cmd tea.Cmd
					m.moveInput, cmd = m.moveInput.Update(msg)
					return m, cmd
				}
			}

			// Otherwise, handle directory tree navigation
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.movingNote = false
				return m, nil

			case "up", "k":
				if m.moveCursor > 0 {
					m.moveCursor--
				}
				return m, nil

			case "down", "j":
				if m.moveCursor < len(m.moveDirTree)-1 {
					m.moveCursor++
				}
				return m, nil

			case "right", "l":
				// Expand directory
				m.toggleDirectoryExpansion()
				return m, nil

			case "left", "h":
				// Collapse directory
				if m.moveCursor < len(m.moveDirTree) {
					node := m.moveDirTree[m.moveCursor]
					if node.isExpanded {
						node.isExpanded = false
						root := m.findRootNode()
						m.moveDirTree = m.flattenDirectoryTree(root)
					}
				}
				return m, nil

			case "n":
				// Create new directory
				m.moveCreatingNewDir = true
				m.moveInput.SetValue("")
				m.moveInput.Focus()
				return m, textinput.Blink

			case "enter":
				// Move note to selected directory
				if m.moveCursor < len(m.moveDirTree) && m.moveTargetIdx < len(m.filteredNotes) {
					node := m.moveDirTree[m.moveCursor]
					note := m.filteredNotes[m.moveTargetIdx]
					// Move the note
					if err := m.notesService.MoveNote(note.FilePath, node.fullPath); err != nil {
						m.err = err
					} else {
						m.loadNotes()
					}
				}
				m.movingNote = false
				return m, nil
			}
			return m, nil
		}

		// Handle new item menu (note or category)
		if m.showingNewMenu {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.showingNewMenu = false
				return m, nil

			case "up", "k":
				if m.newMenuCursor > 0 {
					m.newMenuCursor--
				}
				return m, nil

			case "down", "j":
				if m.newMenuCursor < 1 { // 0: New Note, 1: New Category
					m.newMenuCursor++
				}
				return m, nil

			case "enter", "l":
				m.showingNewMenu = false
				if m.newMenuCursor == 0 {
					// Show template selection for new note
					m.showingTemplates = true
					m.templateCursor = 0
				} else {
					// Show category input
					m.creatingCategory = true
					m.categoryInput.Focus()
					return m, textinput.Blink
				}
				return m, nil
			}
			return m, nil
		}

		// Handle template selection mode
		if m.showingTemplates {
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit

			case "esc":
				m.showingTemplates = false
				return m, nil

			case "up", "k":
				if m.templateCursor > 0 {
					m.templateCursor--
				}
				return m, nil

			case "down", "j":
				if m.templateCursor < len(m.templates)-1 {
					m.templateCursor++
				}
				return m, nil

			case "enter", "l":
				if len(m.templates) > 0 && m.templateCursor < len(m.templates) {
					selectedTemplate := m.templates[m.templateCursor]
					m.showingTemplates = false
					return m, func() tea.Msg {
						return CreateNoteFromTemplateMsg{
							templatePath: selectedTemplate.FilePath,
							targetPath:   m.currentPath,
						}
					}
				}
				return m, nil
			}
			return m, nil
		}

		// Handle search mode
		if m.filterMode == FilterSearch && m.searchInput.Focused() {
			switch msg.String() {
			case "esc":
				m.searchInput.Blur()
				m.filterMode = FilterNone
				m.filteredNotes = m.notes
				m.searchInput.SetValue("")
				return m, nil

			case "enter":
				m.searchInput.Blur()
				query := m.searchInput.Value()
				if query != "" {
					notes, err := m.notesService.SearchNotes(query)
					if err == nil {
						m.filteredNotes = notes
						m.cursor = 0
					}
				}
				return m, nil

			default:
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		}

		// Normal navigation mode
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "esc", "h":
			// If we're in a subdirectory, go up one level
			if m.currentPath != "" {
				m.currentPath = filepath.Dir(m.currentPath)
				if m.currentPath == "." {
					m.currentPath = ""
				}
				m.loadNotes()
				return m, nil
			}
			// Otherwise, go back to dashboard
			return m, func() tea.Msg {
				return BackToDashboardMsg{}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			totalItems := len(m.directories) + len(m.filteredNotes)
			if m.cursor < totalItems-1 {
				m.cursor++
			}
			return m, nil

		case "enter", "l":
			// Check if selecting a directory
			if m.cursor < len(m.directories) {
				// Navigate into directory
				dirName := m.directories[m.cursor]
				if m.currentPath == "" {
					m.currentPath = dirName
				} else {
					m.currentPath = filepath.Join(m.currentPath, dirName)
				}
				m.loadNotes()
				return m, nil
			}
			// Selecting a note
			noteIdx := m.cursor - len(m.directories)
			if noteIdx >= 0 && noteIdx < len(m.filteredNotes) {
				note := m.filteredNotes[noteIdx]
				return m, func() tea.Msg {
					return OpenNoteMsg{filePath: note.FilePath}
				}
			}
			return m, nil

		case "n":
			// Show new item menu (note or category)
			m.showingNewMenu = true
			m.newMenuCursor = 0
			return m, nil

		case "m":
			// Move note to different category
			noteIdx := m.cursor - len(m.directories)
			if noteIdx >= 0 && noteIdx < len(m.filteredNotes) {
				m.movingNote = true
				m.moveTargetIdx = noteIdx
				m.moveCursor = 0
				m.moveCreatingNewDir = false
				m.buildMoveDirectoryTree()
				return m, nil
			}
			return m, nil

		case "/":
			// Start search
			m.filterMode = FilterSearch
			m.searchInput.Focus()
			return m, textinput.Blink

		case "t":
			// Show tags
			m.showingTags = true
			m.tagCursor = 0
			return m, nil

		case "c":
			// Clear filter
			m.filterMode = FilterNone
			m.filteredNotes = m.notes
			m.cursor = 0
			m.searchInput.SetValue("")
			return m, nil

		case "r":
			// Refresh list
			m.loadNotes()
			return m, nil

		case "d":
			// Delete note (only if a note is selected, not a directory)
			noteIdx := m.cursor - len(m.directories)
			if noteIdx >= 0 && noteIdx < len(m.filteredNotes) {
				note := m.filteredNotes[noteIdx]
				m.confirmDelete = true
				m.deleteTarget = note.Name
				m.deleteTargetIdx = noteIdx
			}
			return m, nil

		case "p":
			// Preview note in browser (only if a note is selected, not a directory)
			noteIdx := m.cursor - len(m.directories)
			if noteIdx >= 0 && noteIdx < len(m.filteredNotes) {
				note := m.filteredNotes[noteIdx]
				// Read the note content
				content, err := m.notesService.ReadNote(note.FilePath)
				if err == nil {
					go func() {
						_ = m.previewService.PreviewMarkdown(note.FilePath, content)
					}()
				}
			}
			return m, nil
		}
	}

	return m, nil
}

func (m NotesBrowserModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to go back\n", m.err)
	}

	s := notesBrowserTitleStyle.Render("📝 Notes") + "\n\n"

	// Show search bar if in search mode
	if m.filterMode == FilterSearch {
		s += searchBarStyle.Render(m.searchInput.View()) + "\n\n"
	} else if m.filterMode == FilterTag {
		// Show active tag filter
		s += noteTagStyle.Render("Filtered by tag") + "\n\n"
	}

	// Show confirmation dialog if delete is pending
	if m.confirmDelete {
		dialogText := confirmTextStyle.Render(fmt.Sprintf("Delete '%s'?", m.deleteTarget)) + "\n\n"
		dialogText += "  y: yes   n: no   esc: cancel"
		dialog := confirmDialogStyle.Render(dialogText)
		s += dialog + "\n\n"
	}

	// Show category input
	if m.creatingCategory {
		dialogText := confirmTextStyle.Render("Create New Category") + "\n\n"
		dialogText += "Enter category path (e.g., work/projects):\n"
		dialogText += m.categoryInput.View() + "\n\n"
		dialogText += "  enter: create   esc: cancel"
		dialog := confirmDialogStyle.Render(dialogText)
		s += dialog + "\n\n"
	}

	// Show move note directory selection or new dir input
	if m.movingNote {
		noteName := ""
		if m.moveTargetIdx < len(m.filteredNotes) {
			noteName = m.filteredNotes[m.moveTargetIdx].Name
		}

		if m.moveCreatingNewDir {
			// Show text input for creating new directory
			dialogText := confirmTextStyle.Render(fmt.Sprintf("Move '%s' to new directory", noteName)) + "\n\n"
			dialogText += "Enter directory path (e.g., work/projects):\n"
			dialogText += m.moveInput.View() + "\n\n"
			dialogText += "  enter: move   esc: cancel"
			dialog := confirmDialogStyle.Render(dialogText)
			s += dialog + "\n\n"
		} else {
			// Show directory tree selection
			dialogText := confirmTextStyle.Render(fmt.Sprintf("Move '%s' to:", noteName)) + "\n\n"

			if len(m.moveDirTree) == 0 {
				dialogText += "  No directories found.\n"
				dialogText += "  Press 'n' to create a new directory.\n"
			} else {
				for i, node := range m.moveDirTree {
					depth := strings.Count(node.fullPath, string(filepath.Separator))
					indent := strings.Repeat("  ", depth)

					prefix := "  "
					if i == m.moveCursor {
						prefix = "▶ "
					}

					expandIcon := ""
					if node.hasChildren {
						if node.isExpanded {
							expandIcon = "▼ "
						} else {
							expandIcon = "▶ "
						}
					} else {
						expandIcon = "  "
					}

					line := prefix + indent + expandIcon + "📁 " + node.name
					if i == m.moveCursor {
						dialogText += noteSelectedStyle.Render(line) + "\n"
					} else {
						dialogText += line + "\n"
					}
				}
			}

			dialogText += "\n  ↑/k: up • ↓/j: down • →/l: expand • ←/h: collapse • enter: select • n: new dir • esc: cancel"
			dialog := confirmDialogStyle.Render(dialogText)
			s += dialog + "\n\n"
		}
	}

	// Show new item menu
	if m.showingNewMenu {
		menuText := confirmTextStyle.Render("Create New...") + "\n\n"
		options := []string{"Note", "Category"}
		for i, option := range options {
			if i == m.newMenuCursor {
				menuText += noteSelectedStyle.Render("▶ "+option) + "\n"
			} else {
				menuText += "  " + option + "\n"
			}
		}
		menuText += "\n  enter/l: select   esc: cancel"
		dialog := confirmDialogStyle.Render(menuText)
		s += dialog + "\n\n"
	}

	// Show tag selection overlay
	if m.showingTags {
		s += tagListStyle.Render(m.renderTagList()) + "\n\n"
	} else if m.showingTemplates {
		// Show template selection overlay
		s += tagListStyle.Render(m.renderTemplateList()) + "\n\n"
	} else {
		// Show current path breadcrumb
		if m.currentPath != "" {
			s += noteTagStyle.Render("📁 "+m.currentPath) + "\n\n"
		}

		// Show directories and notes list
		totalItems := len(m.directories) + len(m.filteredNotes)
		if totalItems == 0 {
			s += "  No notes or folders found.\n\n"
			s += noteItemStyle.Render("Press 'n' to create a new note or category") + "\n"
		} else {
			// Render directories first
			for i, dir := range m.directories {
				var line string
				if i == m.cursor {
					line = "▶ 📁 " + dir + "/"
					s += noteSelectedStyle.Render(line) + "\n"
				} else {
					line = "  📁 " + dir + "/"
					s += noteItemStyle.Render(line) + "\n"
				}
			}

			// Then render notes
			for i, note := range m.filteredNotes {
				itemIdx := len(m.directories) + i
				var line string
				if itemIdx == m.cursor {
					line = "▶ " + note.Name
					if len(note.Tags) > 0 {
						line += " " + noteTagStyle.Render("["+strings.Join(note.Tags, ", ")+"]")
					}
					s += noteSelectedStyle.Render(line) + "\n"
				} else {
					line = "  " + note.Name
					if len(note.Tags) > 0 {
						line += " " + noteTagStyle.Render("["+strings.Join(note.Tags, ", ")+"]")
					}
					s += noteItemStyle.Render(line) + "\n"
				}
			}
		}
	}

	s += "\n"

	// Help text
	if m.showingTags {
		s += helpStyle.Render("↑/k: up • ↓/j: down • enter/l: select tag • esc: back")
	} else if m.showingTemplates {
		s += helpStyle.Render("↑/k: up • ↓/j: down • enter/l: select template • esc: back")
	} else if m.showingNewMenu {
		s += helpStyle.Render("↑/k: up • ↓/j: down • enter/l: select • esc: back")
	} else if m.creatingCategory || m.movingNote {
		s += helpStyle.Render("enter: confirm • esc: cancel")
	} else if m.filterMode == FilterSearch && m.searchInput.Focused() {
		s += helpStyle.Render("enter: search • esc: cancel")
	} else {
		s += helpStyle.Render("↑/k: up • ↓/j: down • enter/l: open • p: preview • n: new • m: move • /: search • t: tags • c: clear filter • r: refresh • d: delete • esc/h: back • q: quit")
	}

	// Fill the screen
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height)
		return style.Render(s)
	}

	return s
}

func (m NotesBrowserModel) renderTagList() string {
	var s string
	s += "📌 Select a Tag\n\n"

	if len(m.allTags) == 0 {
		s += "  No tags found\n"
	} else {
		for i, tag := range m.allTags {
			if i == m.tagCursor {
				s += noteSelectedStyle.Render("▶ #"+tag) + "\n"
			} else {
				s += "  #" + tag + "\n"
			}
		}
	}

	return s
}

func (m NotesBrowserModel) renderTemplateList() string {
	var s string
	s += "📄 Select a Template\n\n"

	if len(m.templates) == 0 {
		s += "  No templates found\n"
	} else {
		for i, template := range m.templates {
			if i == m.templateCursor {
				s += noteSelectedStyle.Render("▶ "+template.Name) + "\n"
			} else {
				s += "  " + template.Name + "\n"
			}
		}
	}

	return s
}

type OpenNoteMsg struct {
	filePath string
}

type CreateNoteMsg struct{}

type CreateNoteFromTemplateMsg struct {
	templatePath string
	targetPath   string
}

type BackToDashboardMsg struct{}
