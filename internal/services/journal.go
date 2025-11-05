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

// GetJournalDir returns the journal directory path
func (j *JournalService) GetJournalDir() string {
	return j.journalDir
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
// Returns (journalPath, wasCreated, error)
func (j *JournalService) CreateOrOpenJournal(date time.Time) (string, bool, error) {
	if err := j.EnsureJournalDirExists(date); err != nil {
		return "", false, err
	}

	journalPath := j.GetJournalPathForDate(date)

	// Check if file exists
	if _, err := os.Stat(journalPath); os.IsNotExist(err) {
		// Create new journal entry with header
		header := fmt.Sprintf("# Journal Entry - %s\n\n## Tasks\n\n- \n", date.Format("Monday, January 2, 2006"))
		if err := os.WriteFile(journalPath, []byte(header), 0644); err != nil {
			return "", false, fmt.Errorf("failed to create journal file: %w", err)
		}
		return journalPath, true, nil // Was newly created
	}

	return journalPath, false, nil // Already existed
}

// DeleteJournal deletes a journal file
func (j *JournalService) DeleteJournal(filePath string) error {
	return os.Remove(filePath)
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

// GetWeekBoundaries returns the start (Sunday) and end (Saturday) dates for a given date's week
func (j *JournalService) GetWeekBoundaries(date time.Time) (start time.Time, end time.Time) {
	// Find the Sunday of this week
	start = date
	for start.Weekday() != time.Sunday {
		start = start.AddDate(0, 0, -1)
	}
	// Set to midnight
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())

	// End is 6 days after start (Saturday)
	end = start.AddDate(0, 0, 6)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())

	return start, end
}

// ExtractTasksSection extracts the ## Tasks section from journal content
func (j *JournalService) ExtractTasksSection(content string) string {
	lines := strings.Split(content, "\n")
	var taskLines []string
	inTasksSection := false

	for _, line := range lines {
		// Check if we're entering the Tasks section
		if strings.HasPrefix(strings.TrimSpace(line), "## Tasks") {
			inTasksSection = true
			continue
		}

		// Check if we hit another ## heading (end of Tasks section)
		if inTasksSection && strings.HasPrefix(strings.TrimSpace(line), "##") {
			break
		}

		// Collect lines in Tasks section
		if inTasksSection {
			trimmed := strings.TrimSpace(line)
			// Only include non-empty lines or lines that start with bullet points
			if trimmed != "" || (len(taskLines) > 0 && strings.HasPrefix(trimmed, "-")) {
				taskLines = append(taskLines, line)
			}
		}
	}

	return strings.Join(taskLines, "\n")
}

// GenerateWeeklySummary generates a weekly summary by combining all journal entries for a week
func (j *JournalService) GenerateWeeklySummary(weekStart time.Time) (string, error) {
	// Ensure weekStart is actually a Sunday
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	weekEnd := weekStart.AddDate(0, 0, 6)

	// Build the summary header
	summary := fmt.Sprintf("# Weekly Summary: %s - %s\n\n",
		weekStart.Format("January 2, 2006"),
		weekEnd.Format("January 2, 2006"))

	// First pass: collect all tasks for the combined list
	var allTasks []string
	type dayTasks struct {
		day   time.Time
		tasks string
	}
	var dailyTasks []dayTasks

	for day := weekStart; !day.After(weekEnd); day = day.AddDate(0, 0, 1) {
		journalPath := j.GetJournalPathForDate(day)

		// Check if journal exists for this day
		if _, err := os.Stat(journalPath); os.IsNotExist(err) {
			continue
		}

		// Read the journal content
		content, err := os.ReadFile(journalPath)
		if err != nil {
			continue // Skip days we can't read
		}

		// Extract tasks section
		tasks := j.ExtractTasksSection(string(content))
		tasks = strings.TrimSpace(tasks)

		if tasks != "" {
			// Store for daily breakdown
			dailyTasks = append(dailyTasks, dayTasks{
				day:   day,
				tasks: tasks,
			})

			// Extract individual task lines for combined list
			taskLines := strings.Split(tasks, "\n")
			for _, line := range taskLines {
				// Preserve indentation but check trimmed version for content
				trimmed := strings.TrimSpace(line)
				// Only include non-empty lines that start with a bullet or dash
				if trimmed != "" && (strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*")) {
					// Keep original line with indentation
					allTasks = append(allTasks, line)
				}
			}
		}
	}

	// Generate combined task list
	if len(allTasks) > 0 {
		summary += "## All Tasks (Combined)\n\n"
		for _, task := range allTasks {
			summary += task + "\n"
		}
		summary += "\n"
	}

	// Generate daily breakdown
	if len(dailyTasks) > 0 {
		summary += "## Daily Breakdown\n\n"
		for _, dt := range dailyTasks {
			summary += fmt.Sprintf("### %s\n\n", dt.day.Format("Monday, January 2"))
			summary += dt.tasks + "\n\n"
		}
	}

	if len(allTasks) == 0 {
		summary += "*No tasks recorded this week.*\n"
	}

	// Save the summary to disk
	if err := j.SaveWeeklySummary(weekStart, summary); err != nil {
		// Log error but don't fail - we can still return the summary
		fmt.Fprintf(os.Stderr, "Warning: failed to save weekly summary: %v\n", err)
	}

	return summary, nil
}

// GetWeeklySummaryPath returns the path for a weekly summary file
func (j *JournalService) GetWeeklySummaryPath(weekStart time.Time) string {
	// Ensure weekStart is a Sunday
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	year := weekStart.Format("2006")
	// Use the Sunday date as the filename
	filename := fmt.Sprintf("week-%s.md", weekStart.Format("2006-01-02"))

	return filepath.Join(j.journalDir, "summaries", year, filename)
}

// SaveWeeklySummary saves a weekly summary to disk
func (j *JournalService) SaveWeeklySummary(weekStart time.Time, summary string) error {
	summaryPath := j.GetWeeklySummaryPath(weekStart)
	summaryDir := filepath.Dir(summaryPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(summaryDir, 0755); err != nil {
		return fmt.Errorf("failed to create summary directory: %w", err)
	}

	// Write the summary file
	if err := os.WriteFile(summaryPath, []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write summary file: %w", err)
	}

	return nil
}

// ListWeeklySummaries returns a list of all saved weekly summaries
func (j *JournalService) ListWeeklySummaries() ([]WeeklySummaryInfo, error) {
	var summaries []WeeklySummaryInfo
	summariesDir := filepath.Join(j.journalDir, "summaries")

	// Check if summaries directory exists
	if _, err := os.Stat(summariesDir); os.IsNotExist(err) {
		return summaries, nil // No summaries yet
	}

	// Walk through the summaries directory
	err := filepath.Walk(summariesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Only process .md files
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}

		// Parse the filename to get the week start date
		// Format: week-2006-01-02.md
		filename := filepath.Base(path)
		if !strings.HasPrefix(filename, "week-") {
			return nil
		}

		dateStr := strings.TrimPrefix(filename, "week-")
		dateStr = strings.TrimSuffix(dateStr, ".md")

		weekStart, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil // Skip files with invalid dates
		}

		weekEnd := weekStart.AddDate(0, 0, 6)

		summaries = append(summaries, WeeklySummaryInfo{
			WeekStart: weekStart,
			WeekEnd:   weekEnd,
			FilePath:  path,
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error listing summaries: %w", err)
	}

	// Sort summaries by week start date (newest first)
	for i := 0; i < len(summaries)-1; i++ {
		for j := i + 1; j < len(summaries); j++ {
			if summaries[i].WeekStart.Before(summaries[j].WeekStart) {
				summaries[i], summaries[j] = summaries[j], summaries[i]
			}
		}
	}

	return summaries, nil
}

// WeeklySummaryInfo contains metadata about a saved weekly summary
type WeeklySummaryInfo struct {
	WeekStart time.Time
	WeekEnd   time.Time
	FilePath  string
}

// ReadWeeklySummary reads a saved weekly summary from disk
func (j *JournalService) ReadWeeklySummary(weekStart time.Time) (string, error) {
	summaryPath := j.GetWeeklySummaryPath(weekStart)

	content, err := os.ReadFile(summaryPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no summary exists for week starting %s", weekStart.Format("2006-01-02"))
		}
		return "", fmt.Errorf("failed to read summary: %w", err)
	}

	return string(content), nil
}

// HasJournalEntriesForWeek checks if there are any journal entries for the given week
func (j *JournalService) HasJournalEntriesForWeek(weekStart time.Time) bool {
	// Ensure weekStart is actually a Sunday
	for weekStart.Weekday() != time.Sunday {
		weekStart = weekStart.AddDate(0, 0, -1)
	}

	weekEnd := weekStart.AddDate(0, 0, 6)

	// Iterate through each day of the week
	for day := weekStart; !day.After(weekEnd); day = day.AddDate(0, 0, 1) {
		journalPath := j.GetJournalPathForDate(day)

		// Check if journal exists for this day
		if _, err := os.Stat(journalPath); err == nil {
			return true // Found at least one entry
		}
	}

	return false // No entries found for this week
}
