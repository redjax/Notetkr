package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/services"
)

var (
	cleanMenuItemStyle = lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(lipgloss.Color("240"))

	cleanSelectedItemStyle = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(lipgloss.Color("170")).
				Bold(true)
)

// CleanMenuApp represents the clean menu TUI
type CleanMenuApp struct {
	cfg            *config.Config
	cursor         int
	selected       int
	options        []cleanOption
	cleanupService *services.CleanupService
	spinner        spinner.Model
	running        bool
	done           bool
	stats          *services.CleanupStats
	err            error
}

type cleanOption struct {
	name        string
	description string
	command     string
}

// NewCleanMenuApp creates a new clean menu app
func NewCleanMenuApp(cfg *config.Config) *CleanMenuApp {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return &CleanMenuApp{
		cfg:            cfg,
		cursor:         0,
		cleanupService: services.NewCleanupService(cfg.NotesDir, cfg.JournalDir),
		spinner:        s,
		options: []cleanOption{
			{
				name:        "Clean Images",
				description: "Remove unused images and deduplicate duplicates",
				command:     "images",
			},
			{
				name:        "Exit",
				description: "Return to terminal",
				command:     "exit",
			},
		},
	}
}

func (m *CleanMenuApp) Init() tea.Cmd {
	return nil
}

func (m *CleanMenuApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If cleanup is done, allow exiting
		if m.done {
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			}
			return m, nil
		}

		// If cleanup is running, only allow ctrl+c to cancel
		if m.running {
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
			return m, nil
		}

		// Menu navigation
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}

		case "enter":
			selected := m.options[m.cursor]
			if selected.command == "exit" {
				return m, tea.Quit
			}
			if selected.command == "images" {
				// Start the cleanup
				m.running = true
				return m, tea.Batch(
					m.spinner.Tick,
					m.runCleanup,
				)
			}
		}

	case cleanupCompleteMsg:
		m.running = false
		m.done = true
		m.stats = msg.stats
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if m.running {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *CleanMenuApp) runCleanup() tea.Msg {
	stats, err := m.cleanupService.CleanImages()
	return cleanupCompleteMsg{stats: stats, err: err}
}

func (m *CleanMenuApp) View() string {
	// If cleanup is done, show results
	if m.done {
		s := titleStyle.Render("🧹 Image Cleanup") + "\n\n"

		if m.err != nil {
			s += errorStyle.Render("❌ Cleanup failed!") + "\n\n"
			s += statusStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
		} else {
			s += successStyle.Render("✓ Cleanup completed successfully!") + "\n\n"
			s += renderStats(m.stats)
		}
		s += "\n" + helpStyle.Render("press any key to exit")
		return s
	}

	// If cleanup is running, show spinner
	if m.running {
		s := titleStyle.Render("🧹 Image Cleanup") + "\n\n"
		s += fmt.Sprintf("%s %s\n\n", m.spinner.View(), statusStyle.Render("Cleaning up images..."))
		s += statusStyle.Render("Please wait...") + "\n\n"
		s += helpStyle.Render("ctrl+c: cancel")
		return s
	}

	// Show menu
	s := titleStyle.Render("🧹 Cleanup Menu") + "\n\n"
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
		"Select a cleanup task to run:") + "\n\n"

	for i, option := range m.options {
		cursor := "  "
		style := cleanMenuItemStyle

		if m.cursor == i {
			cursor = "▶ "
			style = cleanSelectedItemStyle
		}

		s += fmt.Sprintf("%s%s\n", cursor, style.Render(option.name))
		if m.cursor == i {
			s += lipgloss.NewStyle().
				PaddingLeft(4).
				Foreground(lipgloss.Color("240")).
				Italic(true).
				Render(option.description) + "\n"
		}
		s += "\n"
	}

	s += "\n" + helpStyle.Render("↑/↓: navigate • enter: select • q/esc: quit")

	return s
}
