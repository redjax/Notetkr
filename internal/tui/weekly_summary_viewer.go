package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type WeeklySummaryViewerModel struct {
	journalService *services.JournalService
	summary        string
	weekStart      time.Time
	weekEnd        time.Time
	width          int
	height         int
	scrollOffset   int
	err            error
}

var (
	weeklySummaryViewerTitleStyle = lipgloss.NewStyle().
					Bold(true).
					Foreground(lipgloss.Color("205")).
					Padding(1, 0)

	weeklySummaryContentStyle = lipgloss.NewStyle().
					Padding(0, 2)
)

func NewWeeklySummaryViewer(journalService *services.JournalService, summary string, weekStart, weekEnd time.Time) WeeklySummaryViewerModel {
	return WeeklySummaryViewerModel{
		journalService: journalService,
		summary:        summary,
		weekStart:      weekStart,
		weekEnd:        weekEnd,
	}
}

func NewWeeklySummaryViewerWithSize(journalService *services.JournalService, summary string, weekStart, weekEnd time.Time, width, height int) WeeklySummaryViewerModel {
	m := NewWeeklySummaryViewer(journalService, summary, weekStart, weekEnd)
	m.width = width
	m.height = height
	return m
}

func (m WeeklySummaryViewerModel) Init() tea.Cmd {
	return nil
}

func (m WeeklySummaryViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case WeeklySummaryGeneratedMsg:
		// Update with regenerated summary
		m.summary = msg.summary
		m.weekStart = msg.weekStart
		m.weekEnd = msg.weekEnd
		return m, nil

	case WeeklySummaryErrorMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc", "h", "left":
			// Return to weekly summary menu
			return NewWeeklySummaryMenuWithSize(m.journalService, m.width, m.height), nil
		case "r":
			// Regenerate summary
			return m, m.regenerateSummary

		case "up", "k":
			// Scroll up
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}

		case "down", "j":
			// Scroll down
			lines := strings.Split(m.summary, "\n")
			maxScroll := len(lines) - (m.height - 6) // Account for header/footer
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset < maxScroll {
				m.scrollOffset++
			}

		case "g":
			// Go to top
			m.scrollOffset = 0

		case "G":
			// Go to bottom
			lines := strings.Split(m.summary, "\n")
			maxScroll := len(lines) - (m.height - 6)
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.scrollOffset = maxScroll
		}
	}

	return m, nil
}

func (m WeeklySummaryViewerModel) regenerateSummary() tea.Msg {
	summary, err := m.journalService.GenerateWeeklySummary(m.weekStart)
	if err != nil {
		return WeeklySummaryErrorMsg{err: err}
	}

	return WeeklySummaryGeneratedMsg{
		summary:   summary,
		weekStart: m.weekStart,
		weekEnd:   m.weekEnd,
	}
}

func (m WeeklySummaryViewerModel) View() string {
	if m.err != nil {
		errMsg := fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to return to menu\n", m.err)
		if m.width > 0 && m.height > 0 {
			style := lipgloss.NewStyle().
				Width(m.width).
				Height(m.height).
				AlignHorizontal(lipgloss.Center).
				AlignVertical(lipgloss.Center)
			return style.Render(errMsg)
		}
		return errMsg
	}

	s := weeklySummaryViewerTitleStyle.Render("📊 Weekly Summary") + "\n"
	s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		fmt.Sprintf("  Week: %s - %s",
			m.weekStart.Format("Jan 2, 2006"),
			m.weekEnd.Format("Jan 2, 2006"))) + "\n\n"

	// Calculate visible content area
	headerLines := 4 // Title + week info + blank lines
	footerLines := 2 // Help text + blank line
	visibleLines := m.height - headerLines - footerLines
	if visibleLines < 1 {
		visibleLines = 10 // Default if height not set
	}

	// Split summary into lines and apply scrolling
	lines := strings.Split(m.summary, "\n")
	startLine := m.scrollOffset
	endLine := startLine + visibleLines

	if endLine > len(lines) {
		endLine = len(lines)
	}
	if startLine > len(lines) {
		startLine = len(lines)
	}

	visibleContent := strings.Join(lines[startLine:endLine], "\n")

	// Display summary content - let it expand to full width
	contentWidth := m.width - 4 // Account for padding
	if contentWidth < 40 {
		contentWidth = 80 // Default width
	}

	contentStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(contentWidth)

	s += contentStyle.Render(visibleContent) + "\n"

	// Scroll indicator
	if len(lines) > visibleLines {
		scrollInfo := fmt.Sprintf("  [Lines %d-%d of %d]", startLine+1, endLine, len(lines))
		s += lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(scrollInfo) + "\n"
	} else {
		s += "\n"
	}

	// Help
	s += helpStyle.Render("↑/k ↓/j: scroll • g/G: top/bottom • r: regenerate • esc/h: back • q: quit")

	// Fill the screen
	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height)
		return style.Render(s)
	}

	return s
}
