package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type SavedSummariesBrowserModel struct {
	journalService *services.JournalService
	summaries      []services.WeeklySummaryInfo
	cursor         int
	width          int
	height         int
	err            error
}

var (
	savedSummariesTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("205")).
					Padding(1, 0)

	savedSummariesItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	savedSummariesSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("170")).
					Bold(true).
					Underline(true)
)

func NewSavedSummariesBrowser(journalService *services.JournalService) SavedSummariesBrowserModel {
	m := SavedSummariesBrowserModel{
		journalService: journalService,
		cursor:         0,
	}
	m.loadSummaries()
	return m
}

func (m *SavedSummariesBrowserModel) loadSummaries() {
	summaries, err := m.journalService.ListWeeklySummaries()
	if err != nil {
		m.err = err
		return
	}
	m.summaries = summaries
}

func (m SavedSummariesBrowserModel) Init() tea.Cmd {
	return nil
}

func (m SavedSummariesBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case WeeklySummaryGeneratedMsg:
		// Switch to weekly summary viewer
		return NewWeeklySummaryViewer(m.journalService, msg.summary, msg.weekStart, msg.weekEnd), nil

	case WeeklySummaryErrorMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc", "h", "left":
			// Return to weekly summary menu
			return NewWeeklySummaryMenu(m.journalService), nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.summaries)-1 {
				m.cursor++
			}

		case "enter", "l", "right", " ":
			if len(m.summaries) == 0 {
				return m, nil
			}

			// Read and display the selected summary
			selectedSummary := m.summaries[m.cursor]
			return m, func() tea.Msg {
				content, err := os.ReadFile(selectedSummary.FilePath)
				if err != nil {
					return WeeklySummaryErrorMsg{err: err}
				}

				return WeeklySummaryGeneratedMsg{
					summary:   string(content),
					weekStart: selectedSummary.WeekStart,
					weekEnd:   selectedSummary.WeekEnd,
				}
			}

		case "d":
			// Delete selected summary
			if len(m.summaries) == 0 {
				return m, nil
			}

			selectedSummary := m.summaries[m.cursor]
			return m, func() tea.Msg {
				if err := os.Remove(selectedSummary.FilePath); err != nil {
					return WeeklySummaryErrorMsg{err: err}
				}
				// Reload the list
				return ReloadSavedSummariesMsg{}
			}
		}

	case ReloadSavedSummariesMsg:
		m.loadSummaries()
		if m.cursor >= len(m.summaries) && len(m.summaries) > 0 {
			m.cursor = len(m.summaries) - 1
		}
		return m, nil
	}

	return m, nil
}

func (m SavedSummariesBrowserModel) View() string {
	s := savedSummariesTitleStyle.Render("📚 Saved Weekly Summaries") + "\n\n"

	if m.err != nil {
		s += fmt.Sprintf("  Error: %v\n\n", m.err)
		s += helpStyle.Render("esc/h: back • q: quit")

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

	if len(m.summaries) == 0 {
		s += "  No saved summaries found.\n\n"
		s += "  Generate a weekly summary to create one!\n\n"
		s += helpStyle.Render("esc/h: back • q: quit")

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

	// Display summaries
	for i, summary := range m.summaries {
		label := fmt.Sprintf("%s - %s",
			summary.WeekStart.Format("Jan 2, 2006"),
			summary.WeekEnd.Format("Jan 2, 2006"))

		cursor := "  "
		if m.cursor == i {
			cursor = "→ "
			s += cursor + savedSummariesSelectedStyle.Render(label) + "\n"
		} else {
			s += cursor + savedSummariesItemStyle.Render(label) + "\n"
		}
	}

	s += "\n" + helpStyle.Render("↑/k ↓/j: navigate • enter/l: view • d: delete • esc/h: back • q: quit")

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

type ReloadSavedSummariesMsg struct{}
