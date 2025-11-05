package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

var (
	spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// CleanImagesApp handles the image cleanup process
type CleanImagesApp struct {
	cleanupService *services.CleanupService
	spinner        spinner.Model
	status         string
	stats          *services.CleanupStats
	err            error
	done           bool
}

type cleanupCompleteMsg struct {
	stats           *services.CleanupStats
	notesDeleted    int
	journalsDeleted int
	err             error
}

// NewCleanImagesApp creates a new image cleanup app
func NewCleanImagesApp(cleanupService *services.CleanupService) *CleanImagesApp {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return &CleanImagesApp{
		cleanupService: cleanupService,
		spinner:        s,
		status:         "Initializing cleanup...",
	}
}

func (m *CleanImagesApp) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.runCleanup,
	)
}

func (m *CleanImagesApp) runCleanup() tea.Msg {
	stats, err := m.cleanupService.CleanImages()
	return cleanupCompleteMsg{stats: stats, err: err}
}

func (m *CleanImagesApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.done {
			// Allow quit after completion
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			}
		} else {
			// Only allow forced quit during processing
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			}
		}

	case cleanupCompleteMsg:
		m.done = true
		m.stats = msg.stats
		m.err = msg.err
		return m, nil

	case spinner.TickMsg:
		if !m.done {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	}

	return m, nil
}

func (m *CleanImagesApp) View() string {
	s := titleStyle.Render("🧹 Image Cleanup") + "\n\n"

	if !m.done {
		s += fmt.Sprintf("%s %s\n\n", m.spinner.View(), statusStyle.Render(m.status))
		s += statusStyle.Render("Processing...") + "\n\n"
		s += helpStyle.Render("ctrl+c: cancel")
	} else {
		if m.err != nil {
			s += errorStyle.Render("❌ Cleanup failed!") + "\n\n"
			s += statusStyle.Render(fmt.Sprintf("Error: %v", m.err)) + "\n"
		} else {
			s += successStyle.Render("✓ Cleanup completed successfully!") + "\n\n"
			s += renderStats(m.stats)
		}
		s += "\n" + helpStyle.Render("press any key to exit")
	}

	return s
}

func renderStats(stats *services.CleanupStats) string {
	if stats == nil {
		return ""
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2)

	content := fmt.Sprintf("Unused images deleted:    %d\n", stats.UnusedImagesDeleted)
	content += fmt.Sprintf("Duplicate images deleted:  %d\n", stats.DuplicateImagesDeleted)
	content += fmt.Sprintf("References updated:        %d\n", stats.ReferencesUpdated)
	content += fmt.Sprintf("Space freed:               %s\n", formatBytes(stats.BytesFreed))

	return style.Render(content)
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
