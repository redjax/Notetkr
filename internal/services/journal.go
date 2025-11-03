package services

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JournalService handles journal-related operations
type JournalService struct {
	journalDir string
}

// NewJournalService creates a new journal service
func NewJournalService(journalDir string) *JournalService {
	return &JournalService{
		journalDir: journalDir,
	}
}

// GetTodayJournalPath returns the path for today's journal entry
func (j *JournalService) GetTodayJournalPath() string {
	return j.GetJournalPathForDate(time.Now())
}

// GetJournalPathForDate returns the journal path for a specific date
func (j *JournalService) GetJournalPathForDate(date time.Time) string {
	year := date.Format("2006")
	month := date.Format("01")
	weekNum := getWeekOfMonth(date)
	filename := date.Format("2006-01-02.md")

	return filepath.Join(j.journalDir, year, month, fmt.Sprintf("Week%d", weekNum), filename)
}

// getWeekOfMonth calculates which week of the month the date falls in
// Weeks start on Sunday
func getWeekOfMonth(date time.Time) int {
	// Get the first day of the month
	firstDay := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())

	// Find the first Sunday of the month (or if 1st is Sunday, that's week 1)
	firstSunday := firstDay
	for firstSunday.Weekday() != time.Sunday {
		firstSunday = firstSunday.AddDate(0, 0, 1)
	}

	// If the date is before the first Sunday, it's in Week 1
	if date.Before(firstSunday) {
		return 1
	}

	// Calculate days since first Sunday
	daysSinceFirstSunday := int(date.Sub(firstSunday).Hours() / 24)

	// Week number is (days since first Sunday / 7) + 1
	weekNum := (daysSinceFirstSunday / 7) + 1

	return weekNum
}

// EnsureJournalDirExists creates the journal directory structure if it doesn't exist
func (j *JournalService) EnsureJournalDirExists(date time.Time) error {
	journalPath := j.GetJournalPathForDate(date)
	journalDir := filepath.Dir(journalPath)

	if err := os.MkdirAll(journalDir, 0755); err != nil {
		return fmt.Errorf("failed to create journal directory: %w", err)
	}

	return nil
}

// CreateOrOpenJournal creates or opens today's journal file
func (j *JournalService) CreateOrOpenJournal(date time.Time) (string, error) {
	if err := j.EnsureJournalDirExists(date); err != nil {
		return "", err
	}

	journalPath := j.GetJournalPathForDate(date)

	// Check if file exists
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		// Create new journal entry with header
		header := fmt.Sprintf("# Journal Entry - %s\n\n## Tasks\n\n- \n\n", date.Format("Monday, January 2, 2006"))
		if err := os.WriteFile(journalPath, []byte(header), 0644); err != nil {
			return "", fmt.Errorf("failed to create journal file: %w", err)
		}
	}

	return journalPath, nil
}

// ReadJournal reads the contents of a journal file
func (j *JournalService) ReadJournal(date time.Time) (string, error) {
	journalPath := j.GetJournalPathForDate(date)

	content, err := os.ReadFile(journalPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no journal entry exists for %s", date.Format("2006-01-02"))
		}
		return "", fmt.Errorf("failed to read journal: %w", err)
	}

	return string(content), nil
}

// WriteJournal writes content to a journal file
func (j *JournalService) WriteJournal(date time.Time, content string) error {
	if err := j.EnsureJournalDirExists(date); err != nil {
		return err
	}

	journalPath := j.GetJournalPathForDate(date)

	if err := os.WriteFile(journalPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write journal: %w", err)
	}

	return nil
}

// JournalEntry represents a journal entry with metadata
type JournalEntry struct {
	Date     time.Time
	FilePath string
	Preview  string
}

// SearchJournals searches all journal entries by content
func (j *JournalService) SearchJournals(query string) ([]JournalEntry, error) {
	var results []JournalEntry

	if query == "" {
		return results, nil
	}

	query = strings.ToLower(query)

	// Walk through all journal files
	err := filepath.Walk(j.journalDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Only process .md files
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip files we can't read
		}

		contentStr := string(content)

		// Check if content contains query
		if strings.Contains(strings.ToLower(contentStr), query) {
			// Try to parse date from filename (format: YYYY-MM-DD.md)
			filename := filepath.Base(path)
			dateStr := strings.TrimSuffix(filename, ".md")
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				// If we can't parse the date, skip this entry
				return nil
			}

			// Create preview (first 100 chars or first line)
			preview := contentStr
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			if idx := strings.Index(preview, "\n"); idx > 0 && idx < 100 {
				preview = preview[:idx] + "..."
			}

			results = append(results, JournalEntry{
				Date:     date,
				FilePath: path,
				Preview:  preview,
			})
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error searching journals: %w", err)
	}

	return results, nil
}
