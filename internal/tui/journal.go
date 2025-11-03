package tui

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type JournalModel struct {
	journalService *services.JournalService
	date           time.Time
	content        string
	filePath       string
	width          int
	height         int
	err            error
}

var (
	journalTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(1, 0)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)
)

func NewJournalModel(journalService *services.JournalService) JournalModel {
	return JournalModel{
		journalService: journalService,
		date:           time.Now(),
	}
}

func (m JournalModel) Init() tea.Cmd {
	return m.loadJournal
}

func (m JournalModel) loadJournal() tea.Msg {
	filePath, err := m.journalService.CreateOrOpenJournal(m.date)
	if err != nil {
		return JournalErrorMsg{err: err}
	}

	content, err := m.journalService.ReadJournal(m.date)
	if err != nil {
		return JournalErrorMsg{err: err}
	}

	return JournalLoadedMsg{
		filePath: filePath,
		content:  content,
	}
}

func (m JournalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case JournalLoadedMsg:
		m.filePath = msg.filePath
		m.content = msg.content
		return m, nil

	case JournalErrorMsg:
		m.err = msg.err
		return m, nil

	case WeeklySummaryGeneratedMsg:
		// Switch to weekly summary viewer
		return NewWeeklySummaryViewer(m.journalService, msg.summary, msg.weekStart, msg.weekEnd), nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			// Return to dashboard
			return NewDashboard(), nil

		case "e":
			// Open in editor
			return m, m.openInEditor

		case "r":
			// Reload journal
			return m, m.loadJournal

		case "g":
			// Open weekly summary menu
			return NewWeeklySummaryMenu(m.journalService), nil
		}
	}

	return m, nil
}

func (m JournalModel) openInEditor() tea.Msg {
	// Get editor from environment, default to vi
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	// Create command to open editor
	cmd := exec.Command(editor, m.filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run editor
	if err := cmd.Run(); err != nil {
		return JournalErrorMsg{err: fmt.Errorf("failed to open editor: %w", err)}
	}

	// Reload journal after editing
	content, err := m.journalService.ReadJournal(m.date)
	if err != nil {
		return JournalErrorMsg{err: err}
	}

	return JournalLoadedMsg{
		filePath: m.filePath,
		content:  content,
	}
}

func (m JournalModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to return to dashboard\n", m.err)
	}

	if m.filePath == "" {
		return "\n  Loading journal...\n"
	}

	s := journalTitleStyle.Render(fmt.Sprintf("📔 Journal Entry - %s", m.date.Format("Monday, January 2, 2006"))) + "\n"
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(fmt.Sprintf("  File: %s", m.filePath)) + "\n\n"

	// Display content
	s += contentStyle.Render(m.content) + "\n\n"

	// Help
	s += helpStyle.Render("e: edit in $EDITOR • r: reload • g: weekly summary • esc: back to dashboard • q: quit")

	return s
}

type JournalLoadedMsg struct {
	filePath string
	content  string
}

type JournalErrorMsg struct {
	err error
}
