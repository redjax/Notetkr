package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type JournalBrowserModel struct {
	journalService   *services.JournalService
	journalDir       string
	breadcrumb       []string // Track navigation path: ["2025", "10", "15"]
	items            []string
	cursor           int
	width            int
	height           int
	err              error
	confirmDelete    bool
	deleteTarget     string
	deleteTargetPath string
	creatingNew      bool
	nameInput        textinput.Model
}

var (
	breadcrumbStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Padding(0, 0, 1, 0)

	itemStyle = lipgloss.NewStyle().
			PaddingLeft(0)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true).
			PaddingLeft(0)

	browserTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(1, 0)

	confirmDialogStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("196")).
				Padding(1, 2).
				Foreground(lipgloss.Color("196"))

	confirmTextStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196"))
)

func NewJournalBrowser(journalService *services.JournalService, journalDir string, width, height int) JournalBrowserModel {
	nameInput := textinput.New()
	nameInput.Placeholder = "Enter journal filename (e.g., 2025-11-05)..."
	nameInput.CharLimit = 100
	nameInput.Width = 50

	m := JournalBrowserModel{
		journalService: journalService,
		journalDir:     journalDir,
		breadcrumb:     []string{},
		cursor:         0,
		width:          width,
		height:         height,
		nameInput:      nameInput,
	}
	m.loadItems()
	return m
}

func (m *JournalBrowserModel) loadItems() {
	m.items = []string{}
	m.cursor = 0

	// Add "Today's Journal" only at root level (no breadcrumb)
	if len(m.breadcrumb) == 0 {
		m.items = append(m.items, "📔 Today's Journal")
	}

	// Build current path
	currentPath := m.journalDir
	for _, part := range m.breadcrumb {
		currentPath = filepath.Join(currentPath, part)
	}

	// Read directory
	entries, err := os.ReadDir(currentPath)
	if err != nil {
		m.err = err
		return
	}

	// Collect items
	var dirs []string
	var files []string

	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	// Sort
	sort.Strings(dirs)                               // Directories A-Z
	sort.Sort(sort.Reverse(sort.StringSlice(files))) // Files Z-A (newest first)

	// Add folders first (years, months, days)
	for _, dir := range dirs {
		m.items = append(m.items, "📁 "+dir)
	}

	// Add journal files
	for _, file := range files {
		m.items = append(m.items, "📄 "+strings.TrimSuffix(file, ".md"))
	}
}

func (m JournalBrowserModel) Init() tea.Cmd {
	return nil
}

func (m JournalBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Handle filename input for new journal
		if m.creatingNew {
			switch msg.String() {
			case "esc":
				m.creatingNew = false
				m.nameInput.Blur()
				m.nameInput.SetValue("")
				return m, nil

			case "enter":
				filename := strings.TrimSpace(m.nameInput.Value())
				if filename == "" {
					return m, nil
				}

				// Ensure .md extension
				if !strings.HasSuffix(filename, ".md") {
					filename = filename + ".md"
				}

				m.creatingNew = false
				m.nameInput.Blur()
				m.nameInput.SetValue("")

				// Build the full path including breadcrumb
				currentPath := m.journalDir
				for _, part := range m.breadcrumb {
					currentPath = filepath.Join(currentPath, part)
				}
				fullPath := filepath.Join(currentPath, filename)

				// Return message to create and open the journal with this filepath
				return m, func() tea.Msg {
					return CreateJournalWithNameMsg{filepath: fullPath}
				}

			default:
				var cmd tea.Cmd
				m.nameInput, cmd = m.nameInput.Update(msg)
				return m, cmd
			}
		}

		// Handle delete confirmation
		if m.confirmDelete {
			switch msg.String() {
			case "y", "Y":
				// Confirm delete
				if err := os.RemoveAll(m.deleteTargetPath); err != nil {
					m.err = fmt.Errorf("failed to delete: %w", err)
				}
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteTargetPath = ""
				m.loadItems()
				return m, nil

			case "n", "N", "esc":
				// Cancel delete
				m.confirmDelete = false
				m.deleteTarget = ""
				m.deleteTargetPath = ""
				return m, nil
			}
			return m, nil
		}

		// Normal navigation
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "n":
			// Show filename input for new journal
			m.creatingNew = true
			m.nameInput.Focus()
			return m, textinput.Blink

		case "esc", "h", "left":
			// Go back/up one level
			if len(m.breadcrumb) > 0 {
				m.breadcrumb = m.breadcrumb[:len(m.breadcrumb)-1]
				m.loadItems()
			} else {
				// Return to dashboard
				return m, func() tea.Msg {
					return BackToDashboardMsg{}
				}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}

		case "d":
			// Delete current item (with confirmation)
			if len(m.items) == 0 || m.cursor >= len(m.items) {
				return m, nil
			}

			selected := m.items[m.cursor]

			// Can't delete "Today's Journal" entry
			if strings.HasPrefix(selected, "📔") {
				return m, nil
			}

			// Build path to delete
			currentPath := m.journalDir
			for _, part := range m.breadcrumb {
				currentPath = filepath.Join(currentPath, part)
			}

			var targetPath string
			if strings.HasPrefix(selected, "📁") {
				// Deleting a folder
				folderName := strings.TrimPrefix(selected, "📁 ")
				targetPath = filepath.Join(currentPath, folderName)
			} else if strings.HasPrefix(selected, "📄") {
				// Deleting a file
				fileName := strings.TrimPrefix(selected, "📄 ")
				targetPath = filepath.Join(currentPath, fileName+".md")
			}

			if targetPath != "" {
				m.confirmDelete = true
				m.deleteTarget = selected
				m.deleteTargetPath = targetPath
			}

		case "g":
			// Open weekly summary menu
			return m, func() tea.Msg {
				return OpenWeeklySummaryMenuMsg{}
			}

		case "enter", "l", "right", " ":
			if len(m.items) == 0 {
				return m, nil
			}

			selected := m.items[m.cursor]

			// Handle "Today's Journal"
			if strings.HasPrefix(selected, "📔") {
				return m, func() tea.Msg {
					return OpenJournalMsg{date: time.Now()}
				}
			}

			// Handle folders
			if strings.HasPrefix(selected, "📁") {
				folderName := strings.TrimPrefix(selected, "📁 ")
				m.breadcrumb = append(m.breadcrumb, folderName)
				m.loadItems()
			}

			// Handle journal files
			if strings.HasPrefix(selected, "📄") {
				// Parse date from filename (YYYY-MM-DD)
				fileName := strings.TrimPrefix(selected, "📄 ")

				// Check if it's a weekly summary file (week-YYYY-MM-DD)
				if strings.HasPrefix(fileName, "week-") {
					// Open weekly summary in editor
					currentPath := m.journalDir
					for _, part := range m.breadcrumb {
						currentPath = filepath.Join(currentPath, part)
					}
					filePath := filepath.Join(currentPath, fileName+".md")

					return m, func() tea.Msg {
						return OpenWeeklySummaryFileMsg{filePath: filePath}
					}
				}

				// Regular journal file - try to parse date
				date, err := time.Parse("2006-01-02", fileName)
				if err == nil {
					return m, func() tea.Msg {
						return OpenJournalMsg{date: date}
					}
				}
			}
		}
	}

	return m, nil
}

func (m JournalBrowserModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to go back\n", m.err)
	}

	s := browserTitleStyle.Render("📚 Journals") + "\n"

	// Show breadcrumb
	if len(m.breadcrumb) > 0 {
		path := "Journals"
		for _, part := range m.breadcrumb {
			path += " > " + part
		}
		s += breadcrumbStyle.Render(path) + "\n"
	}
	s += "\n"

	// Show filename input if creating new journal
	if m.creatingNew {
		inputBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("170")).
			Padding(1, 2).
			Render("📝 New Journal Entry\n\n" + m.nameInput.View() + "\n\nEnter: create • Esc: cancel")
		s += inputBox + "\n\n"
	}

	// Show confirmation dialog if delete is pending
	if m.confirmDelete {
		dialogText := confirmTextStyle.Render(fmt.Sprintf("Delete '%s'?", m.deleteTarget)) + "\n\n"
		dialogText += "  y: yes   n: no   esc: cancel"
		dialog := confirmDialogStyle.Render(dialogText)
		s += dialog + "\n\n"
	}

	// Show items
	if len(m.items) == 0 {
		s += "  No journals found.\n"
	} else {
		for i, item := range m.items {
			if i == m.cursor {
				s += selectedStyle.Render("▶ "+item) + "\n"
			} else {
				s += itemStyle.Render("  "+item) + "\n"
			}
		}
	}

	s += "\n" + helpStyle.Render("n: new entry • ↑/k: up • ↓/j: down • enter/l: open • esc/h: back • g: weekly summary • d: delete • q: quit")

	// Fill the screen
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height)
		return style.Render(s)
	}

	return s
}

type OpenJournalMsg struct {
	date time.Time
}

type OpenJournalEditorMsg struct {
	date time.Time
}

type CreateJournalWithNameMsg struct {
	filepath string
}

type BackToJournalBrowserMsg struct{}

type OpenWeeklySummaryMenuMsg struct{}

type OpenWeeklySummaryFileMsg struct {
	filePath string
}
