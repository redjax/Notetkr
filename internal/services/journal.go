package services

import (
	"fmt"
	"os"
	"path/filepath"
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
	day := date.Format("02")
	filename := date.Format("2006-01-02.md")

	return filepath.Join(j.journalDir, year, month, day, filename)
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
		header := fmt.Sprintf("# Journal Entry - %s\n\n", date.Format("Monday, January 2, 2006"))
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
