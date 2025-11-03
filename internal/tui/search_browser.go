package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/redjax/notetkr/internal/services"
)

type SearchResult struct {
	Type     string // "note" or "journal"
	Name     string
	FilePath string
	Date     string // For journals
	Preview  string
}

type SearchBrowserModel struct {
	journalService *services.JournalService
	notesService   *services.NotesService
	searchInput    textinput.Model
	results        []SearchResult
	cursor         int
	width          int
	height         int
	err            error
	searching      bool
	hasSearched    bool
}

var (
	searchBrowserTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				Padding(1, 0)

	searchInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("170")).
				Padding(0, 1)

	searchResultStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	searchSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	searchTypeNoteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("99")).
				Bold(true)

	searchTypeJournalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("212")).
				Bold(true)

	searchPreviewStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Italic(true)

	searchHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)

func NewSearchBrowser(journalService *services.JournalService, notesService *services.NotesService, width, height int) SearchBrowserModel {
	searchInput := textinput.New()
	searchInput.Placeholder = "Search notes and journals..."
	searchInput.CharLimit = 100
	searchInput.Focus()

	return SearchBrowserModel{
		journalService: journalService,
		notesService:   notesService,
		searchInput:    searchInput,
		width:          width,
		height:         height,
		searching:      false,
		hasSearched:    false,
	}
}

func (m SearchBrowserModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *SearchBrowserModel) performSearch() tea.Msg {
	query := m.searchInput.Value()
	if query == "" {
		return SearchCompletedMsg{results: []SearchResult{}}
	}

	var results []SearchResult

	// Search notes
	notes, err := m.notesService.SearchNotes(query)
	if err == nil {
		for _, note := range notes {
			results = append(results, SearchResult{
				Type:     "note",
				Name:     note.Name,
				FilePath: note.FilePath,
				Preview:  strings.Join(note.Tags, ", "),
			})
		}
	}

	// Search journals
	journals, err := m.journalService.SearchJournals(query)
	if err == nil {
		for _, journal := range journals {
			results = append(results, SearchResult{
				Type:     "journal",
				Name:     journal.Date.Format("Monday, January 2, 2006"),
				FilePath: journal.FilePath,
				Date:     journal.Date.Format("2006-01-02"),
				Preview:  journal.Preview,
			})
		}
	}

	// Sort results: journals first (by date desc), then notes (alphabetically)
	sort.Slice(results, func(i, j int) bool {
		if results[i].Type == results[j].Type {
			if results[i].Type == "journal" {
				return results[i].Date > results[j].Date
			}
			return results[i].Name < results[j].Name
		}
		return results[i].Type == "journal"
	})

	return SearchCompletedMsg{results: results}
}

func (m SearchBrowserModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case SearchCompletedMsg:
		m.results = msg.results
		m.searching = false
		m.hasSearched = true
		m.cursor = 0
		return m, nil

	case tea.KeyMsg:
		// Always allow going back
		if msg.String() == "esc" || msg.String() == "q" {
			return m, func() tea.Msg {
				return BackToDashboardMsg{}
			}
		}

		// If we're focused on search input
		if m.searchInput.Focused() {
			switch msg.String() {
			case "enter":
				// Perform search
				m.searching = true
				m.hasSearched = false
				m.searchInput.Blur()
				return m, m.performSearch

			case "down":
				// Move to results if we have any
				if len(m.results) > 0 {
					m.searchInput.Blur()
				}
				return m, nil

			default:
				// Update search input
				m.searchInput, cmd = m.searchInput.Update(msg)
				return m, cmd
			}
		} else {
			// We're in results list
			switch msg.String() {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				} else {
					// Go back to search input
					m.searchInput.Focus()
					return m, textinput.Blink
				}

			case "down", "j":
				if m.cursor < len(m.results)-1 {
					m.cursor++
				}

			case "enter", " ":
				// Open selected result
				if len(m.results) > 0 {
					result := m.results[m.cursor]
					if result.Type == "note" {
						return m, func() tea.Msg {
							return OpenNoteMsg{filePath: result.FilePath}
						}
					} else {
						// Parse date and open journal
						date, err := parseDate(result.Date)
						if err == nil {
							return m, func() tea.Msg {
								return OpenJournalMsg{date: date}
							}
						}
					}
				}

			case "/":
				// Go back to search input
				m.searchInput.Focus()
				return m, textinput.Blink
			}
		}
	}

	return m, nil
}

func (m SearchBrowserModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press 'esc' to go back\n", m.err)
	}

	var b strings.Builder

	// Title
	b.WriteString(searchBrowserTitleStyle.Render("🔍 Search"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(searchInputStyle.Render(m.searchInput.View()))
	b.WriteString("\n\n")

	// Status or results
	if m.searching {
		b.WriteString("Searching...\n")
	} else if m.hasSearched {
		if len(m.results) == 0 {
			b.WriteString("No results found.\n")
		} else {
			b.WriteString(fmt.Sprintf("Found %d result(s):\n\n", len(m.results)))

			// Calculate how many results we can show
			availableHeight := m.height - 12 // Reserve space for header and help
			if availableHeight < 1 {
				availableHeight = 10
			}

			// Show results
			start := m.cursor
			if !m.searchInput.Focused() && m.cursor >= availableHeight {
				start = m.cursor - availableHeight + 1
			}
			end := start + availableHeight
			if end > len(m.results) {
				end = len(m.results)
			}

			for i := start; i < end; i++ {
				result := m.results[i]
				cursor := "  "
				resultLine := ""

				if i == m.cursor && !m.searchInput.Focused() {
					cursor = "▶ "
				}

				// Format based on type
				if result.Type == "note" {
					typeLabel := searchTypeNoteStyle.Render("[Note]")
					resultLine = fmt.Sprintf("%s %s %s", cursor, typeLabel, result.Name)
					if result.Preview != "" {
						resultLine += " " + searchPreviewStyle.Render("("+result.Preview+")")
					}
				} else {
					typeLabel := searchTypeJournalStyle.Render("[Journal]")
					resultLine = fmt.Sprintf("%s %s %s", cursor, typeLabel, result.Name)
					if result.Preview != "" {
						preview := result.Preview
						if len(preview) > 60 {
							preview = preview[:60] + "..."
						}
						resultLine += "\n    " + searchPreviewStyle.Render(preview)
					}
				}

				if i == m.cursor && !m.searchInput.Focused() {
					b.WriteString(searchSelectedStyle.Render(resultLine))
				} else {
					b.WriteString(searchResultStyle.Render(resultLine))
				}
				b.WriteString("\n")
			}
		}
	} else {
		b.WriteString("Enter a search query and press Enter to search.\n")
	}

	b.WriteString("\n")

	// Help text
	var help string
	if m.searchInput.Focused() {
		help = "enter: search • down: results • esc: back to dashboard"
	} else {
		help = "↑/k: up • ↓/j: down • enter: open • /: new search • esc/q: back to dashboard"
	}
	b.WriteString(searchHelpStyle.Render(help))

	content := b.String()

	if m.width > 0 && m.height > 0 {
		style := lipgloss.NewStyle().
			Width(m.width).
			Height(m.height).
			Padding(1, 2)
		return style.Render(content)
	}

	return content
}

type SearchCompletedMsg struct {
	results []SearchResult
}

// parseDate parses a date string in YYYY-MM-DD format
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}
