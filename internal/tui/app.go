package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/services"
)

// AppModel is the main orchestrator that manages different views
type AppModel struct {
	currentView    tea.Model
	journalService *services.JournalService
	notesDir       string
}

// NewAppModel creates a new app model with dashboard as initial view
func NewAppModel(journalDir, notesDir string) AppModel {
	journalService := services.NewJournalService(journalDir)

	return AppModel{
		currentView:    NewDashboard(),
		journalService: journalService,
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
		case "journal":
			m.currentView = NewJournalModel(m.journalService)
			return m, m.currentView.Init()
		case "browse":
			// TODO: Implement notes browser
			return m, nil
		case "new-note":
			// TODO: Implement new note creation
			return m, nil
		}
	}

	m.currentView, cmd = m.currentView.Update(msg)
	return m, cmd
}

func (m AppModel) View() string {
	return m.currentView.View()
}
