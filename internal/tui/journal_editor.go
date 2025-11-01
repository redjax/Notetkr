package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type EditorMode int

const (
	ModeNormal EditorMode = iota
	ModeInsert
)

type JournalEditorModel struct {
	journalService *services.JournalService
	date           time.Time
	filePath       string
	textarea       textarea.Model
	mode           EditorMode
	width          int
	height         int
	err            error
	saved          bool
	saveMsg        string
}

var (
	editorTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(0, 0, 1, 0)

	editorHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	modeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170"))

	normalModeStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("33"))

	saveSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42")).
				Bold(true)

	saveErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)
)

func NewJournalEditor(journalService *services.JournalService, date time.Time) JournalEditorModel {
	ta := textarea.New()
	ta.Placeholder = "Press 'i' to enter insert mode and start writing..."
	ta.Focus() // Keep focused so cursor is visible
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	// Set reasonable defaults that will be overridden by WindowSizeMsg
	ta.SetWidth(80)
	ta.SetHeight(20)

	m := JournalEditorModel{
		journalService: journalService,
		date:           date,
		textarea:       ta,
		mode:           ModeNormal,
		saved:          false,
	}

	return m
}

func (m JournalEditorModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.loadJournal,
	)
}

func (m JournalEditorModel) loadJournal() tea.Msg {
	filePath, err := m.journalService.CreateOrOpenJournal(m.date)
	if err != nil {
		return JournalEditorErrorMsg{err: err}
	}

	content, err := m.journalService.ReadJournal(m.date)
	if err != nil {
		return JournalEditorErrorMsg{err: err}
	}

	return JournalEditorLoadedMsg{
		filePath: filePath,
		content:  content,
	}
}

func (m JournalEditorModel) saveJournal() tea.Msg {
	content := m.textarea.Value()
	err := m.journalService.WriteJournal(m.date, content)
	if err != nil {
		return JournalEditorErrorMsg{err: err}
	}

	return JournalSavedMsg{}
}

func (m JournalEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve space for title and help text
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - 7)
		return m, nil

	case JournalEditorLoadedMsg:
		m.filePath = msg.filePath
		m.textarea.SetValue(msg.content)
		// Reset cursor to start of document
		m.textarea.CursorStart()
		return m, nil

	case JournalEditorErrorMsg:
		m.err = msg.err
		m.saveMsg = ""
		return m, nil

	case JournalSavedMsg:
		m.saved = true
		m.saveMsg = "✓ Saved"
		m.err = nil
		// Clear save message after 2 seconds
		return m, tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return ClearSaveMsg{}
		})

	case ClearSaveMsg:
		m.saveMsg = ""
		return m, nil

	case tea.KeyMsg:
		// Handle mode-specific keys
		if m.mode == ModeNormal {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "q":
				// Return to journal browser
				return m, func() tea.Msg {
					return BackToJournalBrowserMsg{}
				}

			case "i":
				// Enter insert mode
				m.mode = ModeInsert
				return m, nil

			case "a":
				// Enter insert mode and move cursor right (append)
				m.mode = ModeInsert
				// Send right arrow to move cursor forward
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
				return m, cmd

			case "ctrl+s":
				// Save journal (works in both modes)
				m.saved = false
				m.saveMsg = "Saving..."
				return m, m.saveJournal

			// Vim navigation in normal mode - convert to arrow keys
			case "h":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
				return m, cmd

			case "j":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDown})
				return m, cmd

			case "k":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyUp})
				return m, cmd

			case "l":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
				return m, cmd

			case "left", "right", "up", "down":
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd

			case "w":
				// Word forward - use alt+right
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}, Alt: true})
				return m, cmd

			case "b":
				// Word backward - use alt+left
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}, Alt: true})
				return m, cmd

			case "0", "home":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})
				return m, cmd

			case "$", "end":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
				return m, cmd

			case "g":
				// Go to top of document
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
				return m, cmd

			case "G":
				// Go to bottom of document
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlEnd})
				return m, cmd
			}
			// Block all other keys in normal mode (don't pass to textarea)
			return m, nil
		} else {
			// Insert mode - pass everything to textarea
			switch msg.String() {
			case "esc":
				// Return to normal mode
				m.mode = ModeNormal
				return m, nil

			case "ctrl+s":
				// Save journal
				m.saved = false
				m.saveMsg = "Saving..."
				return m, m.saveJournal

			case "ctrl+c":
				return m, tea.Quit
			}
			// Pass all other keys to textarea in insert mode
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}
	}

	// Always update textarea for cursor blinking
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m JournalEditorModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to go back\n", m.err)
	}

	var b strings.Builder

	// Title
	title := fmt.Sprintf("📝 %s", m.date.Format("Monday, January 2, 2006"))
	b.WriteString(editorTitleStyle.Render(title))
	b.WriteString(" ")

	// Mode indicator
	if m.mode == ModeInsert {
		b.WriteString(modeStyle.Render("-- INSERT --"))
	} else {
		b.WriteString(normalModeStyle.Render("-- NORMAL --"))
	}
	b.WriteString("\n")

	// Save status
	if m.saveMsg != "" {
		if strings.Contains(m.saveMsg, "✓") {
			b.WriteString(saveSuccessStyle.Render(m.saveMsg))
		} else {
			b.WriteString(editorHelpStyle.Render(m.saveMsg))
		}
		b.WriteString("\n")
	}

	// Textarea
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")

	// Help - different based on mode
	var help string
	if m.mode == ModeNormal {
		help = "hjkl: move • i/a: insert • 0/$: line start/end • g/G: top/bottom • ctrl+s: save • q: back"
	} else {
		help = "esc: normal mode • ctrl+s: save • ctrl+c: quit"
	}
	b.WriteString(editorHelpStyle.Render(help))

	content := b.String()

	// Fill the screen
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height)
		return style.Render(content)
	}

	return content
}

type JournalEditorLoadedMsg struct {
	filePath string
	content  string
}

type JournalEditorErrorMsg struct {
	err error
}

type JournalSavedMsg struct{}

type ClearSaveMsg struct{}
