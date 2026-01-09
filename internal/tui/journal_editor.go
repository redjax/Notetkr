package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
	wasJustCreated   bool // Track if this journal was created in this session
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

// NewJournalEditorWithFilename creates a new journal editor with a custom filepath
func NewJournalEditorWithFilename(journalService *services.JournalService, filepath string) JournalEditorModel {
	ta := textarea.New()
	ta.Placeholder = "Press 'i' to enter insert mode and start writing..."
	ta.Focus()
	ta.CharLimit = 0
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(20)

	clipboardHandler := utils.NewClipboardImageHandler()
	_ = clipboardHandler.Initialize()

	m := JournalEditorModel{
		journalService:   journalService,
		filePath:         filepath,
		textarea:         ta,
		mode:             ModeNormal,
		saved:            false,
		undoStack:        []undoState{},
		redoStack:        []undoState{},
		lastContent:      "",
		clipboardHandler: clipboardHandler,
		previewService:   services.NewPreviewService(),
		wasJustCreated:   true, // Mark as newly created
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
	var content string
	var err error
	var wasCreated bool

	// If we have a custom filepath, use it directly
	if m.filePath != "" {
		// Create the file with default template if it doesn't exist
		content, err = m.createOrReadCustomJournal()
		if err != nil {
			return JournalEditorErrorMsg{err: err}
		}
		wasCreated = true
	} else {
		// Use the date-based journal
		filePath, created, err := m.journalService.CreateOrOpenJournal(m.date)
		if err != nil {
			return JournalEditorErrorMsg{err: err}
		}
		m.filePath = filePath
		wasCreated = created

		content, err = m.journalService.ReadJournal(m.date)
		if err != nil {
			return JournalEditorErrorMsg{err: err}
		}
	}

	return JournalEditorLoadedMsg{
		filePath:   m.filePath,
		content:    content,
		wasCreated: wasCreated,
	}
}

// createOrReadCustomJournal creates or reads a custom-named journal file
func (m JournalEditorModel) createOrReadCustomJournal() (string, error) {
	// Check if file exists
	content, err := os.ReadFile(m.filePath)
	if err == nil {
		return string(content), nil
	}

	// File doesn't exist, create it with default template
	template := "# Journal Entry\n\n## Tasks\n\n- \n"

	// Ensure directory exists
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// Create the file
	if err := os.WriteFile(m.filePath, []byte(template), 0644); err != nil {
		return "", err
	}

	return template, nil
}

func (m JournalEditorModel) saveJournal() tea.Msg {
	content := m.textarea.Value()

	var err error
	// If we have a custom filepath, write directly to it
	if m.date.IsZero() {
		err = os.WriteFile(m.filePath, []byte(content), 0644)
	} else {
		err = m.journalService.WriteJournal(m.date, content)
	}

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
		m.wasJustCreated = msg.wasCreated // Track if this was newly created
		// Initialize undo stack with the loaded content
		m.lastContent = msg.content
		m.initialContent = msg.content
		// Position cursor appropriately - return a command to do this after SetValue processes
		if msg.wasCreated {
			return m, func() tea.Msg {
				return PositionCursorMsg{}
			}
		}
		return m, nil

	case PositionCursorMsg:
		// This runs after SetValue has been processed
		// For new journals, move to line 4 (the "- " line)
		m.textarea.SetCursor(0) // Start at beginning
		for i := 0; i < 4; i++ {
			m.textarea.CursorDown()
		}
		// Move to end of line (after "- ")
		m.textarea.CursorEnd()
		return m, nil

	case JournalEditorErrorMsg:
		m.err = msg.err
		m.saveMsg = ""
		return m, nil

	case JournalSavedMsg:
		m.saved = true
		m.saveMsg = "✓ Saved"
		m.err = nil

		// Only clear wasJustCreated if user actually made changes from the template
		currentContent := strings.TrimSpace(m.textarea.Value())
		initialTemplate := strings.TrimSpace(m.initialContent)
		if currentContent != initialTemplate {
			m.wasJustCreated = false // User made real changes
		}

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
				// Check if this is a newly created journal (in this session) that is still empty/unchanged
				if m.wasJustCreated && m.isEmpty() {
					// Delete the empty journal file
					if m.filePath != "" {
						_ = m.journalService.DeleteJournal(m.filePath)
					}
					return m, func() tea.Msg {
						return BackToJournalBrowserMsg{}
					}
				}

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
				// Use smart indentation
				return m, m.insertNewLineWithIndent()

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

			case "x":
				// Delete character under cursor (like x in vim)
				m.deleteChar()
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

			case "enter":
				// Smart indentation for lists
				return m, m.insertNewLineWithIndent()

			case "tab":
				// Indent current line
				m.indentCurrentLine()
				return m, nil

			case "shift+tab":
				// Unindent current line
				m.unindentCurrentLine()
				return m, nil

			case "ctrl+c":
				return m, tea.Quit

			default:
				// Pass all other keys to textarea in insert mode
				m.textarea, cmd = m.textarea.Update(msg)
				return m, cmd
			}
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
	currentLineNum := m.textarea.Line()

	// Check bounds
	if currentLineNum >= len(lines) {
		return
	}

	currentLineText := lines[currentLineNum]

	// Move to start of line
	m.textarea.CursorStart()

	// Delete all characters on the line
	for i := 0; i < len(currentLineText); i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	}

	// If this isn't the last line, delete the newline character too
	if currentLineNum < len(lines)-1 {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	} else if currentLineNum > 0 {
		// If it's the last line but not the only line, delete the newline before it
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}

	// Cursor is now at the start of what was the next line (or end of previous line)
	// Track the change after deletion
	m.trackContentChange()
}

// deleteChar deletes the character under the cursor (like x in vim)
func (m *JournalEditorModel) deleteChar() {
	content := m.textarea.Value()
	if content == "" {
		return
	}

	// Save current state before deletion
	m.trackContentChange()

	// Get cursor position
	lineInfo := m.textarea.LineInfo()
	currentLine := m.textarea.Line()
	cursorCol := lineInfo.ColumnOffset

	lines := strings.Split(content, "\n")
	if currentLine >= len(lines) {
		return
	}

	currentLineText := lines[currentLine]

	// If cursor is at or beyond the end of the line, nothing to delete
	if cursorCol >= len(currentLineText) {
		return
	}

	// Use the built-in delete key to remove the character under the cursor
	// This keeps the cursor in place
	m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})

	// Track the change after deletion
	m.trackContentChange()
}

// hasUnsavedChanges checks if the current content differs from the initial/saved content
func (m *JournalEditorModel) hasUnsavedChanges() bool {
	return m.textarea.Value() != m.initialContent
}

// isEmpty checks if the journal content is effectively empty (only whitespace or unchanged from initial)
func (m *JournalEditorModel) isEmpty() bool {
	content := strings.TrimSpace(m.textarea.Value())
	initialContent := strings.TrimSpace(m.initialContent)

	// Empty if no content or content matches initial template
	return content == "" || content == initialContent
}

// getCurrentLineIndentAndPrefix returns the indentation and list prefix of the current line
func (m *JournalEditorModel) getCurrentLineIndentAndPrefix() (string, string) {
	content := m.textarea.Value()
	lines := strings.Split(content, "\n")
	currentLineNum := m.textarea.Line()

	if currentLineNum >= len(lines) {
		return "", ""
	}

	currentLine := lines[currentLineNum]

	// Match leading whitespace
	indentRegex := regexp.MustCompile(`^(\s*)`)
	indentMatch := indentRegex.FindString(currentLine)

	// Match list markers: -, *, +, or numbered lists (1., 2., etc.)
	listRegex := regexp.MustCompile(`^(\s*)([-*+]|\d+\.)\s`)
	if listMatch := listRegex.FindStringSubmatch(currentLine); len(listMatch) >= 3 {
		return listMatch[1], listMatch[2] + " "
	}

	// No list marker, just return indentation
	return indentMatch, ""
}

// insertNewLineWithIndent inserts a new line preserving indentation and list markers
func (m *JournalEditorModel) insertNewLineWithIndent() tea.Cmd {
	// Check if current line is an empty list item (just "- " or similar with no text)
	content := m.textarea.Value()
	lines := strings.Split(content, "\n")
	currentLineNum := m.textarea.Line()

	if currentLineNum < len(lines) {
		currentLine := lines[currentLineNum]
		// Match empty list items: optional whitespace, list marker, optional space, then end of line
		emptyListRegex := regexp.MustCompile(`^\s*[-*+]\s*$`)
		if emptyListRegex.MatchString(currentLine) {
			// Remove the empty list marker and exit list
			lines[currentLineNum] = ""
			newContent := strings.Join(lines, "\n")
			m.textarea.SetValue(newContent)

			// Move cursor to end of now-empty line and insert newline
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m.trackContentChange()
			return cmd
		}
	}

	indent, listPrefix := m.getCurrentLineIndentAndPrefix()

	// Move to end of line and insert newline
	m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// If there was indentation or a list marker, insert it
	if indent != "" || listPrefix != "" {
		prefix := indent + listPrefix
		for _, r := range prefix {
			m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		}
	}

	m.trackContentChange()
	return textarea.Blink
}

// indentCurrentLine adds indentation to the current line (for Tab key)
func (m *JournalEditorModel) indentCurrentLine() {
	// Save current column position
	lineInfo := m.textarea.LineInfo()
	col := lineInfo.ColumnOffset

	// Move to start of current line
	m.textarea.CursorStart()

	// Insert two spaces by simulating keypress
	for i := 0; i < 2; i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	}

	// Restore cursor to its original column position (now shifted right by 2)
	// Move back to start of line first
	m.textarea.CursorStart()
	// Then move to the new position (original + 2 spaces)
	for i := 0; i < col+2; i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
}

// unindentCurrentLine removes indentation from the current line (for Shift+Tab)
func (m *JournalEditorModel) unindentCurrentLine() {
	// Save current column position
	lineInfo := m.textarea.LineInfo()
	col := lineInfo.ColumnOffset

	content := m.textarea.Value()
	lines := strings.Split(content, "\n")
	currentLineNum := m.textarea.Line()

	if currentLineNum >= len(lines) {
		return
	}

	currentLine := lines[currentLineNum]

	// Check how many spaces to remove
	removeCount := 0
	if strings.HasPrefix(currentLine, "  ") {
		removeCount = 2
	} else if strings.HasPrefix(currentLine, " ") {
		removeCount = 1
	} else if strings.HasPrefix(currentLine, "\t") {
		removeCount = 1
	}

	if removeCount == 0 {
		return // Nothing to remove
	}

	// Move to start of line
	m.textarea.CursorStart()

	// Delete the spaces using backspace (which deletes forward at line start)
	for i := 0; i < removeCount; i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyDelete})
	}

	// Restore cursor to its column position (now shifted left)
	newCol := col - removeCount
	if newCol < 0 {
		newCol = 0
	}
	// Move to the new position
	for i := 0; i < newCol; i++ {
		m.textarea, _ = m.textarea.Update(tea.KeyMsg{Type: tea.KeyRight})
	}
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
		help = "hjkl: move • i/a/o: insert • d: delete line • x: delete char • p: preview • 0/$: line start/end • g/G: top/bottom • ctrl+s: save • q: back"
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
	filePath   string
	content    string
	wasCreated bool
}

type JournalEditorErrorMsg struct {
	err error
}

type JournalSavedMsg struct{}

type ClearSaveMsg struct{}

type PositionCursorMsg struct{}
