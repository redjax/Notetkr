package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type WeeklySummaryMenuModel struct {
	journalService *services.JournalService
	cursor         int
	options        []string
	width          int
	height         int
}

var (
	weeklySummaryMenuTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("205")).
					Padding(1, 0)

	weeklySummaryMenuItemStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("86"))

	weeklySummaryMenuSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("170")).
					Bold(true).
					Underline(true)
)

func NewWeeklySummaryMenu(journalService *services.JournalService) WeeklySummaryMenuModel {
	return WeeklySummaryMenuModel{
		journalService: journalService,
		cursor:         0,
		options:        []string{"Current Week", "Browse Past Weeks", "Browse Saved Summaries"},
	}
}

func (m WeeklySummaryMenuModel) Init() tea.Cmd {
	return nil
}

func (m WeeklySummaryMenuModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case WeeklySummaryGeneratedMsg:
		// Switch to weekly summary viewer
		return NewWeeklySummaryViewer(m.journalService, msg.summary, msg.weekStart, msg.weekEnd), nil

	case WeeklySummaryErrorMsg:
		// Could add error display here, for now just stay on menu
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc", "h", "left":
			// Return to journal browser
			return m, func() tea.Msg {
				return BackToJournalBrowserMsg{}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}

		case "enter", "l", "right", " ":
			if m.cursor == 0 {
				// Current Week - generate summary for current week
				return m, m.generateCurrentWeekSummary
			} else if m.cursor == 1 {
				// Browse Past Weeks - go to week browser
				return NewWeekBrowser(m.journalService), nil
			} else {
				// Browse Saved Summaries - go to saved summaries browser
				return NewSavedSummariesBrowser(m.journalService), nil
			}
		}
	}

	return m, nil
}

func (m WeeklySummaryMenuModel) generateCurrentWeekSummary() tea.Msg {
	now := time.Now()
	summary, err := m.journalService.GenerateWeeklySummary(now)
	if err != nil {
		return WeeklySummaryErrorMsg{err: err}
	}

	start, end := m.journalService.GetWeekBoundaries(now)
	return WeeklySummaryGeneratedMsg{
		summary:   summary,
		weekStart: start,
		weekEnd:   end,
	}
}

func (m WeeklySummaryMenuModel) View() string {
	s := weeklySummaryMenuTitleStyle.Render("📊 Weekly Summary Options") + "\n\n"

	for i, option := range m.options {
		cursor := "  "
		if m.cursor == i {
			cursor = "→ "
			s += cursor + weeklySummaryMenuSelectedStyle.Render(option) + "\n"
		} else {
			s += cursor + weeklySummaryMenuItemStyle.Render(option) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/k ↓/j: navigate • enter/l: select • esc/h: back • q: quit")

	// Center the content and fill the screen
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center)
		return style.Render(s)
	}

	return s
}

type WeeklySummaryGeneratedMsg struct {
	summary   string
	weekStart time.Time
	weekEnd   time.Time
}

type WeeklySummaryErrorMsg struct {
	err error
}
