package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/services"
)

// AppModel is the main orchestrator that manages different views
type AppModel struct {
	currentView    tea.Model
	journalService *services.JournalService
	journalDir     string
	notesDir       string
}

// NewAppModel creates a new app model with dashboard as initial view
func NewAppModel(journalDir, notesDir string) AppModel {
	journalService := services.NewJournalService(journalDir)

	return AppModel{
		currentView:    NewDashboard(),
		journalService: journalService,
		journalDir:     journalDir,
		notesDir:       notesDir,
	}
}

// NewJournalBrowserApp creates a new app model starting at the journal browser
func NewJournalBrowserApp(journalDir, notesDir string) AppModel {
	journalService := services.NewJournalService(journalDir)

	return AppModel{
		currentView:    NewJournalBrowser(journalService, journalDir),
		journalService: journalService,
		journalDir:     journalDir,
		notesDir:       notesDir,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle menu selections
	switch msg := msg.(type) {
	case MenuSelectionMsg:
		switch msg.Selection {
		case "today-journal":
			// Open today's journal in editor
			m.currentView = NewJournalEditor(m.journalService, time.Now())
			return m, m.currentView.Init()
		case "journals":
			// Open journal browser
			m.currentView = NewJournalBrowser(m.journalService, m.journalDir)
			return m, m.currentView.Init()
		case "notes":
			// TODO: Implement notes browser
			return m, nil
		}
	case OpenJournalMsg:
		// Open specific journal date in editor
		m.currentView = NewJournalEditor(m.journalService, msg.date)
		return m, m.currentView.Init()
	}

	m.currentView, cmd = m.currentView.Update(msg)
	return m, cmd
}

func (m AppModel) View() string {
	return m.currentView.View()
}
