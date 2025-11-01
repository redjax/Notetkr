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

type NotesEditorModel struct {
	notesService *services.NotesService
	filePath     string
	templatePath string
	textarea     textarea.Model
	mode         EditorMode
	width        int
	height       int
	saveMsg      string
	saved        bool
	err          error
	isNewNote    bool
	noteName     string
}

var (
	notesEditorTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(1, 0)

	notesModeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

	notesNormalModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	notesSaveSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("42"))

	notesHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

// NewNotesEditor creates a new notes editor for an existing note
func NewNotesEditor(notesService *services.NotesService, filePath string) NotesEditorModel {
	ta := textarea.New()
	ta.Placeholder = "Press 'i' to enter insert mode and start writing..."
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(20)

	m := NotesEditorModel{
		notesService: notesService,
		filePath:     filePath,
		textarea:     ta,
		mode:         ModeNormal,
		saved:        false,
		isNewNote:    false,
	}

	return m
}

// NewNotesEditorForNew creates a new notes editor for a new note
func NewNotesEditorForNew(notesService *services.NotesService) NotesEditorModel {
	ta := textarea.New()
	ta.Placeholder = "Enter note name..."
	ta.Focus()
	ta.CharLimit = 100
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(1)

	m := NotesEditorModel{
		notesService: notesService,
		textarea:     ta,
		mode:         ModeInsert, // Start in insert mode for name
		saved:        false,
		isNewNote:    true,
	}

	return m
}

// NewNotesEditorForNewWithTemplate creates a new notes editor for a new note from a template
func NewNotesEditorForNewWithTemplate(notesService *services.NotesService, templatePath string) NotesEditorModel {
	ta := textarea.New()
	ta.Placeholder = "Enter note name..."
	ta.Focus()
	ta.CharLimit = 100
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(1)

	m := NotesEditorModel{
		notesService: notesService,
		templatePath: templatePath,
		textarea:     ta,
		mode:         ModeInsert, // Start in insert mode for name
		saved:        false,
		isNewNote:    true,
	}

	return m
}

func (m NotesEditorModel) Init() tea.Cmd {
	if m.isNewNote {
		return textarea.Blink
	}
	return tea.Batch(
		textarea.Blink,
		m.loadNote,
	)
}

func (m NotesEditorModel) loadNote() tea.Msg {
	content, err := m.notesService.ReadNote(m.filePath)
	if err != nil {
		return NotesEditorErrorMsg{err: err}
	}

	return NotesEditorLoadedMsg{
		filePath: m.filePath,
		content:  content,
	}
}

func (m NotesEditorModel) saveNote() tea.Msg {
	content := m.textarea.Value()
	err := m.notesService.WriteNote(m.filePath, content)
	if err != nil {
		return NotesEditorErrorMsg{err: err}
	}

	return NotesSavedMsg{}
}

func (m NotesEditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.isNewNote {
			m.textarea.SetWidth(msg.Width - 4)
			m.textarea.SetHeight(1)
		} else {
			m.textarea.SetWidth(msg.Width - 4)
			m.textarea.SetHeight(msg.Height - 7)
		}
		return m, nil

	case NotesEditorLoadedMsg:
		m.filePath = msg.filePath
		m.textarea.SetValue(msg.content)
		// Reset cursor to start of document
		m.textarea.CursorStart()
		return m, nil

	case NotesEditorErrorMsg:
		m.err = msg.err
		return m, nil

	case NotesSavedMsg:
		m.saved = true
		m.saveMsg = "✓ Saved"
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return ClearSaveMsg{}
		})

	case ClearSaveMsg:
		m.saveMsg = ""
		return m, nil

	case tea.KeyMsg:
		// Handle new note name entry
		if m.isNewNote {
			switch msg.String() {
			case "esc":
				return m, func() tea.Msg {
					return BackToDashboardMsg{}
				}

			case "enter":
				// Create the note
				noteName := strings.TrimSpace(m.textarea.Value())
				if noteName == "" {
					return m, nil
				}

				var filePath string
				var err error

				// Create from template if one was selected
				if m.templatePath != "" {
					filePath, err = m.notesService.CreateNoteFromTemplate(noteName, m.templatePath)
				} else {
					filePath, err = m.notesService.CreateNote(noteName)
				}

				if err != nil {
					m.err = err
					return m, nil
				}

				// Switch to normal editor mode
				m.isNewNote = false
				m.filePath = filePath
				m.noteName = noteName
				m.mode = ModeNormal
				// Use actual window height if available, otherwise default
				if m.height > 0 {
					m.textarea.SetHeight(m.height - 7)
				} else {
					m.textarea.SetHeight(20)
				}
				m.textarea.CharLimit = 0
				m.textarea.Placeholder = "Press 'i' to enter insert mode and start writing..."

				return m, m.loadNote

			default:
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		}

		// Normal editor mode
		if m.mode == ModeNormal {
			switch msg.String() {
			case "q":
				// Go back to notes browser
				return m, func() tea.Msg {
					return BackToNotesBrowserMsg{}
				}

			case "ctrl+s":
				return m, m.saveNote

			case "i":
				m.mode = ModeInsert
				m.textarea.Focus()
				return m, textarea.Blink

			case "a":
				m.mode = ModeInsert
				m.textarea.Focus()
				// Move cursor to end of line
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
				return m, tea.Batch(cmd, textarea.Blink)

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
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}, Alt: true})
				return m, cmd

			case "b":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{}, Alt: true})
				return m, cmd

			case "0", "home":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})
				return m, cmd

			case "$", "end":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
				return m, cmd

			case "g":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlHome})
				return m, cmd

			case "G":
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlEnd})
				return m, cmd
			}
			return m, nil
		} else {
			// Insert mode
			switch msg.String() {
			case "esc":
				m.mode = ModeNormal
				m.textarea.Blur()
				return m, nil

			case "ctrl+s":
				return m, m.saveNote

			default:
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m NotesEditorModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to go back\n", m.err)
	}

	var b strings.Builder

	if m.isNewNote {
		// New note name entry
		b.WriteString(notesEditorTitleStyle.Render("📝 New Note"))
		b.WriteString("\n\n")
		b.WriteString("Enter note name:\n")
		b.WriteString(m.textarea.View())
		b.WriteString("\n\n")
		b.WriteString(notesHelpStyle.Render("enter: create • esc: cancel"))
	} else {
		// Normal editor
		title := fmt.Sprintf("📝 %s", m.filePath)
		if m.noteName != "" {
			title = fmt.Sprintf("📝 %s", m.noteName)
		}
		b.WriteString(notesEditorTitleStyle.Render(title))
		b.WriteString(" ")

		if m.mode == ModeInsert {
			b.WriteString(notesModeStyle.Render("-- INSERT --"))
		} else {
			b.WriteString(notesNormalModeStyle.Render("-- NORMAL --"))
		}
		b.WriteString("\n")

		if m.saveMsg != "" {
			if strings.Contains(m.saveMsg, "✓") {
				b.WriteString(notesSaveSuccessStyle.Render(m.saveMsg))
			} else {
				b.WriteString(notesHelpStyle.Render(m.saveMsg))
			}
			b.WriteString("\n")
		}

		b.WriteString(m.textarea.View())
		b.WriteString("\n\n")

		var help string
		if m.mode == ModeNormal {
			help = "hjkl: move • i/a: insert • 0/$: line start/end • g/G: top/bottom • ctrl+s: save • q: back to browser"
		} else {
			help = "esc: normal mode • ctrl+s: save • ctrl+c: quit"
		}
		b.WriteString(notesHelpStyle.Render(help))
	}

	content := b.String()

	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height)
		return style.Render(content)
	}

	return content
}

type NotesEditorLoadedMsg struct {
	filePath string
	content  string
}

type NotesEditorErrorMsg struct {
	err error
}

type NotesSavedMsg struct{}

type BackToNotesBrowserMsg struct{}
