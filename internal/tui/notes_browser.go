package tui

import (
	"fmt"
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

type NotesBrowserModel struct {
	notesService     *services.NotesService
	notes            []services.Note
	filteredNotes    []services.Note
	allTags          []string
	templates        []services.Note
	cursor           int
	width            int
	height           int
	err              error
	searchInput      textinput.Model
	filterMode       FilterMode
	showingTags      bool
	showingTemplates bool
	tagCursor        int
	templateCursor   int
	confirmDelete    bool
	deleteTarget     string
	deleteTargetIdx  int
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

	m := NotesBrowserModel{
		notesService:     notesService,
		searchInput:      searchInput,
		filterMode:       FilterNone,
		showingTags:      false,
		showingTemplates: false,
		width:            width,
		height:           height,
	}

	// Initialize default templates
	m.notesService.InitializeDefaultTemplates()

	m.loadNotes()
	m.loadTemplates()
	return m
}

func (m *NotesBrowserModel) loadNotes() {
	notes, err := m.notesService.ListNotes()
	if err != nil {
		m.err = err
		return
	}

	// Sort by modified time (newest first)
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].ModTime.After(notes[j].ModTime)
	})

	m.notes = notes
	m.filteredNotes = notes
	m.cursor = 0

	// Load all tags
	tags, err := m.notesService.GetAllTags()
	if err == nil {
		sort.Strings(tags)
		m.allTags = tags
	}
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
						return CreateNoteFromTemplateMsg{templatePath: selectedTemplate.FilePath}
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
			// Go back to dashboard
			return m, func() tea.Msg {
				return BackToDashboardMsg{}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case "down", "j":
			if m.cursor < len(m.filteredNotes)-1 {
				m.cursor++
			}
			return m, nil

		case "enter", "l":
			if len(m.filteredNotes) > 0 {
				note := m.filteredNotes[m.cursor]
				return m, func() tea.Msg {
					return OpenNoteMsg{filePath: note.FilePath}
				}
			}
			return m, nil

		case "n":
			// Show template selection
			m.showingTemplates = true
			m.templateCursor = 0
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
			// Delete note
			if len(m.filteredNotes) > 0 {
				note := m.filteredNotes[m.cursor]
				m.confirmDelete = true
				m.deleteTarget = note.Name
				m.deleteTargetIdx = m.cursor
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

	// Show tag selection overlay
	if m.showingTags {
		s += tagListStyle.Render(m.renderTagList()) + "\n\n"
	} else if m.showingTemplates {
		// Show template selection overlay
		s += tagListStyle.Render(m.renderTemplateList()) + "\n\n"
	} else {
		// Show notes list
		if len(m.filteredNotes) == 0 {
			s += "  No notes found.\n\n"
			s += noteItemStyle.Render("Press 'n' to create a new note") + "\n"
		} else {
			for i, note := range m.filteredNotes {
				var line string
				if i == m.cursor {
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
	} else if m.filterMode == FilterSearch && m.searchInput.Focused() {
		s += helpStyle.Render("enter: search • esc: cancel")
	} else {
		s += helpStyle.Render("↑/k: up • ↓/j: down • enter/l: open • n: new (from template) • /: search • t: tags • c: clear filter • r: refresh • d: delete • esc/h: back • q: quit")
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
}

type BackToDashboardMsg struct{}
