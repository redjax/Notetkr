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

type undoState struct {
	content string
	line    int
	column  int
}

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
	undoStack    []undoState
	redoStack    []undoState
	lastContent  string
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
		undoStack:    []undoState{},
		redoStack:    []undoState{},
		lastContent:  "",
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
		undoStack:    []undoState{},
		redoStack:    []undoState{},
		lastContent:  "",
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
		undoStack:    []undoState{},
		redoStack:    []undoState{},
		lastContent:  "",
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
		// Initialize undo stack with the loaded content
		m.lastContent = msg.content
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

			case "ctrl+z":
				// Undo
				m.undo()
				return m, nil

			case "ctrl+y":
				// Redo
				m.redo()
				return m, nil

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

			case "d":
				// Delete current line (like dd in vim)
				m.deleteLine()
				return m, nil
			}
			return m, nil
		} else {
			// Insert mode
			switch msg.String() {
			case "esc":
				m.mode = ModeNormal
				return m, nil

			case "ctrl+s":
				return m, m.saveNote

			case "ctrl+z":
				// Undo
				m.undo()
				return m, nil

			case "ctrl+y":
				// Redo
				m.redo()
				return m, nil

			default:
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)

	// Track content changes for undo/redo
	m.trackContentChange()

	return m, cmd
}

// trackContentChange saves the current content to undo stack if it changed
func (m *NotesEditorModel) trackContentChange() {
	currentContent := m.textarea.Value()
	if currentContent != m.lastContent {
		// Get current cursor position
		lineInfo := m.textarea.LineInfo()
		currentLine := m.textarea.Line()

		// Save previous content and cursor position to undo stack
		m.undoStack = append(m.undoStack, undoState{
			content: m.lastContent,
			line:    currentLine,
			column:  lineInfo.ColumnOffset,
		})
		// Limit undo stack size to 100 entries
		if len(m.undoStack) > 100 {
			m.undoStack = m.undoStack[1:]
		}
		m.lastContent = currentContent
		// Clear redo stack on new change
		m.redoStack = []undoState{}
	}
}

// undo restores the previous content
func (m *NotesEditorModel) undo() {
	if len(m.undoStack) == 0 {
		return
	}

	// Get current cursor position
	lineInfo := m.textarea.LineInfo()
	currentLine := m.textarea.Line()

	// Save current content and cursor to redo stack
	m.redoStack = append(m.redoStack, undoState{
		content: m.lastContent,
		line:    currentLine,
		column:  lineInfo.ColumnOffset,
	})

	// Pop from undo stack
	previousState := m.undoStack[len(m.undoStack)-1]
	m.undoStack = m.undoStack[:len(m.undoStack)-1]

	// Restore content
	m.textarea.SetValue(previousState.content)
	m.lastContent = previousState.content

	// Restore cursor position
	// First, move to the correct line
	targetLine := previousState.line
	currentLine = m.textarea.Line()
	for currentLine < targetLine && currentLine < m.textarea.LineCount()-1 {
		m.textarea.CursorDown()
		currentLine++
	}
	for currentLine > targetLine && currentLine > 0 {
		m.textarea.CursorUp()
		currentLine--
	}
	// Then set the column position
	m.textarea.SetCursor(previousState.column)
}

// redo restores content that was undone
func (m *NotesEditorModel) redo() {
	if len(m.redoStack) == 0 {
		return
	}

	// Get current cursor position
	lineInfo := m.textarea.LineInfo()
	currentLine := m.textarea.Line()

	// Save current content and cursor to undo stack
	m.undoStack = append(m.undoStack, undoState{
		content: m.lastContent,
		line:    currentLine,
		column:  lineInfo.ColumnOffset,
	})

	// Pop from redo stack
	nextState := m.redoStack[len(m.redoStack)-1]
	m.redoStack = m.redoStack[:len(m.redoStack)-1]

	// Restore content
	m.textarea.SetValue(nextState.content)
	m.lastContent = nextState.content

	// Restore cursor position
	// First, move to the correct line
	targetLine := nextState.line
	currentLine = m.textarea.Line()
	for currentLine < targetLine && currentLine < m.textarea.LineCount()-1 {
		m.textarea.CursorDown()
		currentLine++
	}
	for currentLine > targetLine && currentLine > 0 {
		m.textarea.CursorUp()
		currentLine--
	}
	// Then set the column position
	m.textarea.SetCursor(nextState.column)
}

// deleteLine deletes the current line where the cursor is positioned
func (m *NotesEditorModel) deleteLine() {
	content := m.textarea.Value()
	if content == "" {
		return
	}

	// Save current state before deletion
	m.trackContentChange()

	// Move to start of line
	m.textarea.Update(tea.KeyMsg{Type: tea.KeyHome})

	// Delete from cursor to end of line (Ctrl+K)
	m.textarea.Update(tea.KeyMsg{Type: tea.KeyCtrlK})

	// Delete the newline character if not on last line
	currentContent := m.textarea.Value()
	if currentContent != "" && m.textarea.Line() < len(strings.Split(currentContent, "\n"))-1 {
		m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	}

	// Track the change after deletion
	m.trackContentChange()
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
			help = "hjkl: move • i/a: insert • d: delete line • 0/$: line start/end • g/G: top/bottom • ctrl+s: save • q: back to browser"
		} else {
			help = "esc: normal mode • ctrl+z/y: undo/redo • ctrl+s: save • ctrl+c: quit"
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
