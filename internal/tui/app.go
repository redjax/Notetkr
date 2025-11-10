package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/services"
)

// AppModel is the main orchestrator that manages different views
type AppModel struct {
	currentView    tea.Model
	journalService *services.JournalService
	notesService   *services.NotesService
	cfg            *config.Config
	journalDir     string
	notesDir       string
	width          int
	height         int
}

// NewAppModel creates a new app model with dashboard as initial view
func NewAppModel(cfg *config.Config) AppModel {
	journalService := services.NewJournalService(cfg.JournalDir)
	notesService := services.NewNotesService(cfg.NotesDir)

	return AppModel{
		currentView:    NewDashboard(),
		journalService: journalService,
		notesService:   notesService,
		cfg:            cfg,
		journalDir:     cfg.JournalDir,
		notesDir:       cfg.NotesDir,
	}
}

// NewJournalBrowserApp creates a new app model starting at the journal browser
func NewJournalBrowserApp(journalDir, notesDir string) AppModel {
	journalService := services.NewJournalService(journalDir)
	notesService := services.NewNotesService(notesDir)

	return AppModel{
		currentView:    NewJournalBrowser(journalService, journalDir, 0, 0),
		journalService: journalService,
		notesService:   notesService,
		journalDir:     journalDir,
		notesDir:       notesDir,
	}
}

// NewNotesBrowserApp creates a new app model starting at the notes browser
func NewNotesBrowserApp(journalDir, notesDir string) AppModel {
	journalService := services.NewJournalService(journalDir)
	notesService := services.NewNotesService(notesDir)

	return AppModel{
		currentView:    NewNotesBrowser(notesService, 0, 0),
		journalService: journalService,
		notesService:   notesService,
		journalDir:     journalDir,
		notesDir:       notesDir,
	}
}

// NewSearchBrowserApp creates a new app model starting at the search browser
func NewSearchBrowserApp(journalDir, notesDir string, query string) AppModel {
	journalService := services.NewJournalService(journalDir)
	notesService := services.NewNotesService(notesDir)

	return AppModel{
		currentView:    NewSearchBrowserWithQuery(journalService, notesService, 0, 0, query),
		journalService: journalService,
		notesService:   notesService,
		journalDir:     journalDir,
		notesDir:       notesDir,
	}
}

// NewTodayJournalApp creates a new app model starting with today's journal open
func NewTodayJournalApp(journalDir, notesDir string) AppModel {
	journalService := services.NewJournalService(journalDir)
	notesService := services.NewNotesService(notesDir)

	return AppModel{
		currentView:    NewJournalEditor(journalService, time.Now()),
		journalService: journalService,
		notesService:   notesService,
		journalDir:     journalDir,
		notesDir:       notesDir,
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.currentView.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Track window size
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	// Handle menu selections
	switch msg := msg.(type) {
	case MenuSelectionMsg:
		switch msg.Selection {
		case "today-journal":
			// Open today's journal in editor
			m.currentView = NewJournalEditor(m.journalService, time.Now())
			// Send window size to new view
			if m.width > 0 && m.height > 0 {
				m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
			}
			return m, tea.Batch(cmd, m.currentView.Init())
		case "journals":
			// Open journal browser
			m.currentView = NewJournalBrowser(m.journalService, m.journalDir, m.width, m.height)
			return m, m.currentView.Init()
		case "notes":
			// Open notes browser
			m.currentView = NewNotesBrowser(m.notesService, m.width, m.height)
			return m, m.currentView.Init()
		case "search":
			// Open search browser
			m.currentView = NewSearchBrowser(m.journalService, m.notesService, m.width, m.height)
			return m, m.currentView.Init()
		case "clean":
			// Open clean menu
			m.currentView = NewCleanMenuAppWithSize(m.cfg, m.width, m.height)
			return m, m.currentView.Init()
		}
	case OpenJournalMsg:
		// Open specific journal date in editor
		m.currentView = NewJournalEditor(m.journalService, msg.date)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case OpenJournalEditorMsg:
		// Open journal editor for specific date
		m.currentView = NewJournalEditor(m.journalService, msg.date)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case CreateJournalWithNameMsg:
		// Create new journal with custom filepath
		m.currentView = NewJournalEditorWithFilename(m.journalService, msg.filepath)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case OpenNoteMsg:
		// Open specific note in editor
		m.currentView = NewNotesEditor(m.notesService, msg.filePath)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case CreateNoteMsg:
		// Create new note
		m.currentView = NewNotesEditorForNew(m.notesService)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case CreateNoteFromTemplateMsg:
		// Create new note from template
		m.currentView = NewNotesEditorForNewWithTemplate(m.notesService, msg.templatePath, msg.targetPath)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case BackToNotesBrowserMsg:
		// Return to notes browser
		m.currentView = NewNotesBrowser(m.notesService, m.width, m.height)
		return m, m.currentView.Init()
	case BackToJournalBrowserMsg:
		// Return to journal browser
		m.currentView = NewJournalBrowser(m.journalService, m.journalDir, m.width, m.height)
		return m, m.currentView.Init()
	case OpenWeeklySummaryMenuMsg:
		// Open weekly summary menu
		m.currentView = NewWeeklySummaryMenu(m.journalService)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case OpenWeeklySummaryFileMsg:
		// Open weekly summary file in editor
		m.currentView = NewNotesEditor(m.notesService, msg.filePath)
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	case BackToDashboardMsg:
		// Return to dashboard
		m.currentView = NewDashboard()
		// Send window size to new view
		if m.width > 0 && m.height > 0 {
			m.currentView, cmd = m.currentView.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
		}
		return m, tea.Batch(cmd, m.currentView.Init())
	}

	m.currentView, cmd = m.currentView.Update(msg)
	return m, cmd
}

func (m AppModel) View() string {
	return m.currentView.View()
}
