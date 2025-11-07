package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type WeekBrowserModel struct {
	journalService *services.JournalService
	currentDate    time.Time // The date we're browsing from
	cursor         int
	weeks          []weekOption
	width          int
	height         int
}

type weekOption struct {
	start time.Time
	end   time.Time
	label string
}

var (
	weekBrowserTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(1, 0)

	weekBrowserItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	weekBrowserSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("170")).
					Bold(true).
					Underline(true)
)

func NewWeekBrowser(journalService *services.JournalService) WeekBrowserModel {
	now := time.Now()
	m := WeekBrowserModel{
		journalService: journalService,
		currentDate:    now,
		cursor:         0,
	}
	m.generateWeekOptions()
	return m
}

func NewWeekBrowserWithSize(journalService *services.JournalService, width, height int) WeekBrowserModel {
	m := NewWeekBrowser(journalService)
	m.width = width
	m.height = height
	return m
}

func (m *WeekBrowserModel) generateWeekOptions() {
	m.weeks = []weekOption{}

	// Keep searching back in time until we find 12 weeks with journal entries
	// or we've searched 52 weeks (1 year)
	weeksFound := 0
	maxWeeksToSearch := 52

	for i := 0; i < maxWeeksToSearch && weeksFound < 12; i++ {
		// Go back i weeks from current date
		weekDate := m.currentDate.AddDate(0, 0, -7*i)
		start, end := m.journalService.GetWeekBoundaries(weekDate)

		// Only include weeks that have at least one journal entry
		if m.journalService.HasJournalEntriesForWeek(start) {
			label := fmt.Sprintf("%s - %s",
				start.Format("Jan 2, 2006"),
				end.Format("Jan 2, 2006"))

			m.weeks = append(m.weeks, weekOption{
				start: start,
				end:   end,
				label: label,
			})
			weeksFound++
		}
	}
}

func (m WeekBrowserModel) Init() tea.Cmd {
	return nil
}

func (m WeekBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case WeeklySummaryGeneratedMsg:
		// Switch to weekly summary viewer
		return NewWeeklySummaryViewerWithSize(m.journalService, msg.summary, msg.weekStart, msg.weekEnd, m.width, m.height), nil

	case WeeklySummaryErrorMsg:
		// Could add error display here, for now just stay on browser
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc", "h", "left":
			// Return to weekly summary menu
			return NewWeeklySummaryMenuWithSize(m.journalService, m.width, m.height), nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.weeks)-1 {
				m.cursor++
			}

		case "enter", "l", "right", " ":
			// Generate summary for selected week
			selectedWeek := m.weeks[m.cursor]
			return m, func() tea.Msg {
				summary, err := m.journalService.GenerateWeeklySummary(selectedWeek.start)
				if err != nil {
					return WeeklySummaryErrorMsg{err: err}
				}
				return WeeklySummaryGeneratedMsg{
					summary:   summary,
					weekStart: selectedWeek.start,
					weekEnd:   selectedWeek.end,
				}
			}
		}
	}

	return m, nil
}

func (m WeekBrowserModel) View() string {
	s := weekBrowserTitleStyle.Render("📅 Select Week") + "\n\n"

	// Check if we have any weeks to display
	if len(m.weeks) == 0 {
		s += "  No weeks with journal entries found.\n\n"
		s += helpStyle.Render("esc: back • q: quit")

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

	// Display weeks
	for i, week := range m.weeks {
		cursor := "  "
		if m.cursor == i {
			cursor = "→ "
			s += cursor + weekBrowserSelectedStyle.Render(week.label) + "\n"
		} else {
			s += cursor + weekBrowserItemStyle.Render(week.label) + "\n"
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
