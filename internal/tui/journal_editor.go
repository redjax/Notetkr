package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
	"github.com/redjax/notetkr/internal/utils"
)

type EditorMode int

const (
	ModeNormal EditorMode = iota
	ModeInsert
)

type JournalEditorModel struct {
	journalService   *services.JournalService
	date             time.Time
	filePath         string
	textarea         textarea.Model
	mode             EditorMode
	width            int
	height           int
	err              error
	saved            bool
	saveMsg          string
	undoStack        []undoState
	redoStack        []undoState
	lastContent      string
	clipboardHandler *utils.ClipboardImageHandler
	showQuitConfirm  bool
	initialContent   string
	previewService   *services.PreviewService
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

	clipboardHandler := utils.NewClipboardImageHandler()
	_ = clipboardHandler.Initialize()

	m := JournalEditorModel{
		journalService:   journalService,
		date:             date,
		textarea:         ta,
		mode:             ModeNormal,
		saved:            false,
		undoStack:        []undoState{},
		redoStack:        []undoState{},
		lastContent:      "",
		clipboardHandler: clipboardHandler,
		previewService:   services.NewPreviewService(),
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
		// Initialize undo stack with the loaded content
		m.lastContent = msg.content
		m.initialContent = msg.content
		return m, nil

	case JournalEditorErrorMsg:
		m.err = msg.err
		m.saveMsg = ""
		return m, nil

	case JournalSavedMsg:
		m.saved = true
		m.saveMsg = "✓ Saved"
		m.err = nil
		m.initialContent = m.textarea.Value()
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
			// Handle quit confirmation dialog
			if m.showQuitConfirm {
				switch msg.String() {
				case "y", "Y":
					// User confirmed quit
					return m, func() tea.Msg {
						return BackToJournalBrowserMsg{}
					}
				case "n", "N", "esc":
					// User cancelled quit
					m.showQuitConfirm = false
					return m, nil
				}
				return m, nil
			}

			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit

			case "q":
				// Check if there are unsaved changes
				if m.hasUnsavedChanges() {
					m.showQuitConfirm = true
					return m, nil
				}
				// No unsaved changes, return to journal browser
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

			case "o":
				// Insert new line below cursor and enter insert mode (like vim)
				m.mode = ModeInsert
				// Move to end of current line, then insert newline
				m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
				m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m.trackContentChange()
				return m, cmd

			case "ctrl+s":
				// Save journal (works in both modes)
				m.saved = false
				m.saveMsg = "Saving..."
				return m, m.saveJournal

			case "ctrl+z":
				// Undo
				m.undo()
				return m, nil

			case "ctrl+y":
				// Redo
				m.redo()
				return m, nil

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

			case "d":
				// Delete current line (like dd in vim)
				m.deleteLine()
				return m, nil

			case "p":
				// Preview markdown in browser
				if m.filePath != "" {
					content := m.textarea.Value()
					go func() {
						_ = m.previewService.PreviewMarkdown(m.filePath, content)
					}()
					m.saveMsg = "✓ Opening preview in browser..."
				}
				return m, nil
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

			case "ctrl+z":
				// Undo
				m.undo()
				return m, nil

			case "ctrl+y":
				// Redo
				m.redo()
				return m, nil

			case "alt+v":
				// Paste image from clipboard
				m.saveMsg = "Checking clipboard for image..."

				if m.clipboardHandler == nil {
					m.saveMsg = "⚠ Clipboard handler not initialized"
					return m, nil
				}

				hasImage := m.clipboardHandler.HasImage()
				if hasImage {
					// Handle image paste
					m.saveMsg = "Image detected, saving..."
					if err := m.pasteImage(); err != nil {
						m.saveMsg = fmt.Sprintf("❌ Error: %v", err)
					} else {
						m.saveMsg = "✓ Image inserted successfully!"
					}
					return m, nil
				}

				// No image in clipboard
				m.saveMsg = "❌ No image in clipboard"
				return m, nil

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

	// Track content changes for undo/redo
	m.trackContentChange()

	return m, cmd
}

// trackContentChange saves the current content to undo stack if it changed
func (m *JournalEditorModel) trackContentChange() {
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
func (m *JournalEditorModel) undo() {
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
func (m *JournalEditorModel) redo() {
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
func (m *JournalEditorModel) deleteLine() {
	content := m.textarea.Value()
	if content == "" {
		return
	}

	// Save current state before deletion
	m.trackContentChange()

	lines := strings.Split(content, "\n")
	lineInfo := m.textarea.Line()
	currentLine := lineInfo

	// Check bounds
	if currentLine >= len(lines) {
		return
	}

	// Remove the current line
	newLines := append(lines[:currentLine], lines[currentLine+1:]...)
	newContent := strings.Join(newLines, "\n")

	// If we deleted the last line and there's content remaining, ensure proper ending
	if currentLine >= len(newLines) && len(newLines) > 0 {
		currentLine = len(newLines) - 1
	}

	// Update content
	m.textarea.SetValue(newContent)

	// Position cursor at the beginning of the same line number (or last line if we deleted the last line)
	// We need to count newlines to position correctly
	if currentLine > 0 {
		targetPos := 0
		for i := 0; i < currentLine; i++ {
			if i < len(newLines) {
				targetPos += len(newLines[i]) + 1 // +1 for newline
			}
		}
		m.textarea.SetCursor(targetPos)
	} else {
		m.textarea.SetCursor(0)
	}

	// Track the change after deletion
	m.trackContentChange()
}

// hasUnsavedChanges checks if the current content differs from the initial/saved content
func (m *JournalEditorModel) hasUnsavedChanges() bool {
	return m.textarea.Value() != m.initialContent
}

// pasteImage handles pasting an image from the clipboard
func (m *JournalEditorModel) pasteImage() error {
	if m.clipboardHandler == nil {
		return fmt.Errorf("clipboard handler not initialized")
	}

	if m.filePath == "" {
		return fmt.Errorf("cannot determine journal location for image attachment")
	}

	// Get the journal directory
	journalDir := m.journalService.GetJournalDir()

	// Use a centralized .attachments/imgs directory
	imgsDir := filepath.Join(journalDir, ".attachments", "imgs")

	// Save the image and get the filename
	filename, err := m.clipboardHandler.SaveClipboardImage(imgsDir, "image")
	if err != nil {
		return err
	}

	// Create the relative path for the markdown link
	relativePath := fmt.Sprintf(".attachments/imgs/%s", filename)

	// Insert the markdown image syntax at cursor position
	// Use angle brackets to handle paths with spaces
	imageMarkdown := fmt.Sprintf("![Pasted image](<%s>)", relativePath)

	// Get current content and cursor position
	content := m.textarea.Value()
	lines := strings.Split(content, "\n")
	lineInfo := m.textarea.LineInfo()
	currentLine := m.textarea.Line()

	// Calculate cursor position from line and column
	cursorPos := 0
	for i := 0; i < currentLine && i < len(lines); i++ {
		cursorPos += len(lines[i]) + 1 // +1 for newline
	}
	cursorPos += lineInfo.ColumnOffset

	// Insert the markdown at cursor position
	newContent := content[:cursorPos] + imageMarkdown + content[cursorPos:]
	m.textarea.SetValue(newContent)

	// Move cursor after the inserted text
	m.textarea.SetCursor(cursorPos + len(imageMarkdown))

	// Track the change
	m.trackContentChange()

	return nil
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

	// Show quit confirmation dialog if needed
	if m.showQuitConfirm {
		confirmStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)
		b.WriteString(confirmStyle.Render("⚠ You have unsaved changes. Quit anyway? (y/n)"))
		b.WriteString("\n\n")
	}

	// Help - different based on mode
	var help string
	if m.mode == ModeNormal {
		help = "hjkl: move • i/a/o: insert • d: delete line • p: preview • 0/$: line start/end • g/G: top/bottom • ctrl+s: save • q: back"
	} else {
		help = "esc: normal mode • ctrl+z/y: undo/redo • alt+v: paste image • ctrl+s: save • ctrl+c: quit"
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
