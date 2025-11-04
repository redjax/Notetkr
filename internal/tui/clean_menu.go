package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/config"
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
	cfg      *config.Config
	cursor   int
	selected int
	options  []cleanOption
}

type cleanOption struct {
	name        string
	description string
	command     string
}

// NewCleanMenuApp creates a new clean menu app
func NewCleanMenuApp(cfg *config.Config) *CleanMenuApp {
	return &CleanMenuApp{
		cfg:    cfg,
		cursor: 0,
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
			// For now, just quit - the command will be handled by cobra
			// In a future iteration, we could launch the cleanup directly
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *CleanMenuApp) View() string {
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
