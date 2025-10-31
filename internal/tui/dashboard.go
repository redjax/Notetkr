package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DashboardModel struct {
	choices  []string
	cursor   int
	selected int
	width    int
	height   int
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			Padding(1, 0)

	menuItemStyle = lipgloss.NewStyle().
			PaddingLeft(4)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true).
				PaddingLeft(2)
)

func NewDashboard() DashboardModel {
	return DashboardModel{
		choices: []string{
			"Today's Journal",
			"Journals",
			"Notes",
			"Quit",
		},
		cursor:   0,
		selected: -1,
	}
}

func (m DashboardModel) Init() tea.Cmd {
	return nil
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.choices)-1 {
				m.cursor++
			}

		case "enter", "l", "right", " ":
			m.selected = m.cursor

			// Handle menu selections
			switch m.selected {
			case 0: // Today's Journal
				return m, func() tea.Msg {
					return MenuSelectionMsg{Selection: "today-journal"}
				}
			case 1: // Journals
				return m, func() tea.Msg {
					return MenuSelectionMsg{Selection: "journals"}
				}
			case 2: // Notes
				return m, func() tea.Msg {
					return MenuSelectionMsg{Selection: "notes"}
				}
			case 3: // Quit
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m DashboardModel) View() string {
	s := titleStyle.Render("📝 Notetkr") + "\n\n"

	for i, choice := range m.choices {
		cursor := "  "
		if m.cursor == i {
			cursor = "▶ "
			s += selectedItemStyle.Render(cursor + choice)
		} else {
			s += menuItemStyle.Render(cursor + choice)
		}
		s += "\n"
	}

	s += "\n" + helpStyle.Render("↑/k: up • ↓/j: down • enter/l: select • q: quit")

	// Center the content and fill the screen
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Align(lipgloss.Center, lipgloss.Center)
		return style.Render(s)
	}

	return s
}

type MenuSelectionMsg struct {
	Selection string
}

var helpStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("241")).
	Padding(1, 0)
