package services

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Note represents a note with metadata
type Note struct {
	Name       string
	FilePath   string
	Tags       []string
	Keywords   []string
	Attendees  []Attendee
	ModTime    time.Time
	IsTemplate bool
}

// Attendee represents a meeting attendee with optional metadata
type Attendee struct {
	Name    string
	Company string
	Email   string
}

type NotesService struct {
	notesDir     string
	templatesDir string
}

func NewNotesService(notesDir string) *NotesService {
	templatesDir := filepath.Join(notesDir, ".templates")
	return &NotesService{
		notesDir:     notesDir,
		templatesDir: templatesDir,
	}
}

// GetNotesDir returns the notes directory path
func (s *NotesService) GetNotesDir() string {
	return s.notesDir
}

// ListNotes returns all notes in the notes directory (excluding templates)
func (s *NotesService) ListNotes() ([]Note, error) {
	var notes []Note

	err := filepath.Walk(s.notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip template directory
		if info.IsDir() && path == s.templatesDir {
			return filepath.SkipDir
		}

		// Only include .md files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, _ := filepath.Rel(s.notesDir, path)

			// Extract metadata from file
			tags, _ := s.extractTags(path)
			keywords, _ := s.extractKeywords(path)
			attendees, _ := s.extractAttendees(path)

			notes = append(notes, Note{
				Name:       relPath,
				FilePath:   path,
				Tags:       tags,
				Keywords:   keywords,
				Attendees:  attendees,
				ModTime:    info.ModTime(),
				IsTemplate: false,
			})
		}
		return nil
	})

	return notes, err
}

// ListTemplates returns all template notes
func (s *NotesService) ListTemplates() ([]Note, error) {
	var templates []Note

	// Ensure templates directory exists
	if err := os.MkdirAll(s.templatesDir, 0755); err != nil {
		return nil, err
	}

	err := filepath.Walk(s.templatesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only include .md files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			relPath, _ := filepath.Rel(s.templatesDir, path)

			templates = append(templates, Note{
				Name:       relPath,
				FilePath:   path,
				ModTime:    info.ModTime(),
				IsTemplate: true,
			})
		}
		return nil
	})

	// Sort by name
	sort.Slice(templates, func(i, j int) bool {
		return templates[i].Name < templates[j].Name
	})

	return templates, err
}

// extractTags reads a note file and extracts tags from the content
// Tags are in the format: #tag or tags: tag1, tag2
func (s *NotesService) extractTags(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	tags := make(map[string]bool)
	text := string(content)

	// Extract frontmatter and body separately
	var frontmatterText string
	var bodyText string

	// Check for frontmatter with --- delimiters
	fmBlockRe := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---\s*\n?(.*)`)
	if fmBlock := fmBlockRe.FindStringSubmatch(text); len(fmBlock) > 2 {
		frontmatterText = fmBlock[1]
		bodyText = fmBlock[2]
	} else {
		// No frontmatter delimiters - treat everything as potential frontmatter for backwards compatibility
		frontmatterText = text
		bodyText = text
	}

	// Extract hashtag-style tags (#tag) from body content only (not frontmatter)
	hashtagRe := regexp.MustCompile(`#([a-zA-Z0-9_-]+)`)
	matches := hashtagRe.FindAllStringSubmatch(bodyText, -1)
	for _, match := range matches {
		if len(match) > 1 {
			tags[strings.ToLower(match[1])] = true
		}
	}

	// Extract tags from frontmatter
	// Only match if there's actual content on the same line (use [ \t] for space/tab only, not \s which includes newline)
	tagsRe := regexp.MustCompile(`(?m)^tags:[ \t]+(\S[^\n]*)$`)
	if tagMatches := tagsRe.FindStringSubmatch(frontmatterText); len(tagMatches) > 1 {
		tagContent := strings.TrimSpace(tagMatches[1])
		if tagContent != "" {
			tagList := strings.Split(tagContent, ",")
			for _, tag := range tagList {
				tag = strings.TrimSpace(tag)
				tag = strings.ToLower(tag)
				if tag != "" {
					tags[tag] = true
				}
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}

	return result, nil
}

// extractKeywords reads a note file and extracts keywords from frontmatter
// Supports both YAML frontmatter with --- delimiters and inline format:
//
//	---
//	keywords: keyword1, keyword2, keyword3
//	---
//
// Or:
//
//	keywords: keyword1, keyword2, keyword3
func (s *NotesService) extractKeywords(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	keywords := make([]string, 0)

	// Extract frontmatter block if it exists
	var frontmatterText string

	// Check for frontmatter with --- delimiters
	fmBlockRe := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	if fmBlock := fmBlockRe.FindStringSubmatch(text); len(fmBlock) > 1 {
		frontmatterText = fmBlock[1]
	} else {
		// Fallback to inline frontmatter format
		frontmatterText = text
	}

	// Extract keywords from frontmatter
	// Only match if there's actual content on the same line (use [ \t] for space/tab only, not \s which includes newline)
	keywordsRe := regexp.MustCompile(`(?m)^keywords:[ \t]+(\S[^\n]*)$`)
	if matches := keywordsRe.FindStringSubmatch(frontmatterText); len(matches) > 1 {
		keywordContent := strings.TrimSpace(matches[1])
		if keywordContent != "" {
			keywordList := strings.Split(keywordContent, ",")
			for _, keyword := range keywordList {
				keyword = strings.TrimSpace(keyword)
				if keyword != "" {
					keywords = append(keywords, keyword)
				}
			}
		}
	}

	return keywords, nil
}

// extractAttendees reads a note file and extracts attendees from YAML frontmatter
// Supports nested YAML structure:
// attendees:
//
//	david correll:
//	jack kenyon:
//	  company: embrace pet insurance
//	  email: jkenyon@example.com
func (s *NotesService) extractAttendees(filePath string) ([]Attendee, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	text := string(content)
	attendees := make([]Attendee, 0)

	// Extract frontmatter block if it exists
	var frontmatterText string

	// Check for frontmatter with --- delimiters
	fmBlockRe := regexp.MustCompile(`(?s)^---\s*\n(.*?)\n---`)
	if fmBlock := fmBlockRe.FindStringSubmatch(text); len(fmBlock) > 1 {
		frontmatterText = fmBlock[1]
	} else {
		// Fallback to inline frontmatter format
		frontmatterText = text
	}

	// Find the attendees section
	attendeesRe := regexp.MustCompile(`(?m)^attendees:\s*$`)
	if !attendeesRe.MatchString(frontmatterText) {
		return attendees, nil
	}

	// Split content into lines
	lines := strings.Split(frontmatterText, "\n")
	inAttendees := false
	var currentAttendee *Attendee

	for _, line := range lines {
		// Check if we're entering the attendees section
		if strings.TrimSpace(line) == "attendees:" {
			inAttendees = true
			continue
		}

		// Exit attendees section if we hit another top-level key
		if inAttendees && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(line, ":") {
			inAttendees = false
			break
		}

		if !inAttendees {
			continue
		}

		// Parse attendee lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		// Check indentation level
		indent := len(line) - len(strings.TrimLeft(line, " \t"))

		// Main attendee (2 spaces indent)
		if indent <= 2 && strings.HasSuffix(trimmed, ":") {
			// Save previous attendee if exists
			if currentAttendee != nil {
				attendees = append(attendees, *currentAttendee)
			}
			// Start new attendee
			name := strings.TrimSuffix(trimmed, ":")
			currentAttendee = &Attendee{Name: name}
		} else if indent > 2 && currentAttendee != nil {
			// Nested properties (4+ spaces indent)
			if strings.Contains(trimmed, ":") {
				parts := strings.SplitN(trimmed, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					switch strings.ToLower(key) {
					case "company":
						currentAttendee.Company = value
					case "email":
						currentAttendee.Email = value
					}
				}
			}
		}
	}

	// Add last attendee
	if currentAttendee != nil {
		attendees = append(attendees, *currentAttendee)
	}

	return attendees, nil
}

// SearchNotes searches notes by name, tags, or content
func (s *NotesService) SearchNotes(query string) ([]Note, error) {
	allNotes, err := s.ListNotes()
	if err != nil {
		return nil, err
	}

	if query == "" {
		return allNotes, nil
	}

	query = strings.ToLower(query)
	var results []Note

	for _, note := range allNotes {
		// Search in filename
		if strings.Contains(strings.ToLower(note.Name), query) {
			results = append(results, note)
			continue
		}

		// Search in tags
		foundInTags := false
		for _, tag := range note.Tags {
			if strings.Contains(tag, query) {
				results = append(results, note)
				foundInTags = true
				break
			}
		}
		if foundInTags {
			continue
		}

		// Search in content
		content, err := os.ReadFile(note.FilePath)
		if err == nil && strings.Contains(strings.ToLower(string(content)), query) {
			results = append(results, note)
		}
	}

	return results, nil
}

// FilterByTag returns notes that have the specified tag
func (s *NotesService) FilterByTag(tag string) ([]Note, error) {
	allNotes, err := s.ListNotes()
	if err != nil {
		return nil, err
	}

	tag = strings.ToLower(tag)
	var results []Note

	for _, note := range allNotes {
		for _, noteTag := range note.Tags {
			if noteTag == tag {
				results = append(results, note)
				break
			}
		}
	}

	return results, nil
}

// GetAllTags returns all unique tags across all notes
func (s *NotesService) GetAllTags() ([]string, error) {
	allNotes, err := s.ListNotes()
	if err != nil {
		return nil, err
	}

	tags := make(map[string]bool)
	for _, note := range allNotes {
		for _, tag := range note.Tags {
			tags[tag] = true
		}
	}

	result := make([]string, 0, len(tags))
	for tag := range tags {
		result = append(result, tag)
	}

	return result, nil
}

// CreateNote creates a new note file
func (s *NotesService) CreateNote(name string) (string, error) {
	return s.CreateNoteInPath(name, "")
}

// CreateNoteInPath creates a new note in a specific subdirectory
func (s *NotesService) CreateNoteInPath(name string, relPath string) (string, error) {
	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	// Build target directory
	targetDir := s.notesDir
	if relPath != "" {
		targetDir = filepath.Join(s.notesDir, relPath)
	}

	filePath := filepath.Join(targetDir, name)

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	// Create file if it doesn't exist
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		// Write initial template with proper YAML frontmatter
		// Don't auto-fill the title - let user add it if they want
		template := "---\ntags:\nkeywords:\n---\n\n# \n\n"
		_, err = file.WriteString(template)
		if err != nil {
			return "", err
		}
	}

	return filePath, nil
}

// ReadNote reads a note's content
func (s *NotesService) ReadNote(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteNote writes content to a note
func (s *NotesService) WriteNote(filePath, content string) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(filePath, []byte(content), 0644)
}

// DeleteNote deletes a note file
func (s *NotesService) DeleteNote(filePath string) error {
	return os.Remove(filePath)
}

// CreateNoteFromTemplate creates a new note using a template
func (s *NotesService) CreateNoteFromTemplate(name, templatePath string) (string, error) {
	return s.CreateNoteFromTemplateInPath(name, templatePath, "")
}

// CreateNoteFromTemplateInPath creates a new note using a template in a specific subdirectory
func (s *NotesService) CreateNoteFromTemplateInPath(name, templatePath, relPath string) (string, error) {
	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	// Build target directory
	targetDir := s.notesDir
	if relPath != "" {
		targetDir = filepath.Join(s.notesDir, relPath)
	}

	filePath := filepath.Join(targetDir, name)

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}

	// Read template content
	templateContent, err := s.ReadNote(templatePath)
	if err != nil {
		return "", err
	}

	// Write template content to new note
	if err := s.WriteNote(filePath, templateContent); err != nil {
		return "", err
	}

	return filePath, nil
}

// CreateTemplate creates a new template file
func (s *NotesService) CreateTemplate(name, content string) (string, error) {
	// Ensure templates directory exists
	if err := os.MkdirAll(s.templatesDir, 0755); err != nil {
		return "", err
	}

	// Ensure .md extension
	if !strings.HasSuffix(name, ".md") {
		name = name + ".md"
	}

	filePath := filepath.Join(s.templatesDir, name)

	// Write template
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// SaveAsTemplate saves an existing note as a template
func (s *NotesService) SaveAsTemplate(sourceNotePath, templateName string) error {
	// Read source note
	content, err := s.ReadNote(sourceNotePath)
	if err != nil {
		return err
	}

	// Create template
	_, err = s.CreateTemplate(templateName, content)
	return err
}

// DeleteTemplate deletes a template file
func (s *NotesService) DeleteTemplate(templatePath string) error {
	return os.Remove(templatePath)
}

// InitializeDefaultTemplates creates default templates if they don't exist
func (s *NotesService) InitializeDefaultTemplates() error {
	// Ensure templates directory exists
	if err := os.MkdirAll(s.templatesDir, 0755); err != nil {
		return err
	}

	// Define default templates with proper YAML frontmatter
	templates := map[string]string{
		"meeting-notes.md": `---
tags: meeting
keywords:
attendees:
  
---

# Meeting Notes

## Notes



## Takeaways

`,
		"todo-list.md": `---
tags: todo
keywords:
---

# TODO List

## Tasks

- [ ] 
- [ ] 
- [ ] 

## Completed

- [x] 
`,
		"blank.md": `---
tags:
keywords:
---

# 
`,
	}

	// Create or update templates
	for name, content := range templates {
		filePath := filepath.Join(s.templatesDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// CreateCategory creates a new directory/category for notes
func (s *NotesService) CreateCategory(categoryPath string) error {
	// Clean the path and build absolute path
	cleanPath := filepath.Clean(categoryPath)
	fullPath := filepath.Join(s.notesDir, cleanPath)

	// Prevent creating paths outside notes directory
	relPath, err := filepath.Rel(s.notesDir, fullPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return filepath.ErrBadPattern
	}

	// Create the directory and any necessary parent directories
	return os.MkdirAll(fullPath, 0755)
}

// MoveNote moves a note to a new location
func (s *NotesService) MoveNote(oldPath, newCategoryPath string) error {
	// Clean the new category path
	cleanPath := filepath.Clean(newCategoryPath)
	targetDir := filepath.Join(s.notesDir, cleanPath)

	// Prevent moving outside notes directory
	relPath, err := filepath.Rel(s.notesDir, targetDir)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return filepath.ErrBadPattern
	}

	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return err
	}

	// Build new file path with same filename
	filename := filepath.Base(oldPath)
	newPath := filepath.Join(targetDir, filename)

	// Move the file
	return os.Rename(oldPath, newPath)
}

// ListNotesInPath returns notes and directories in a specific path
func (s *NotesService) ListNotesInPath(relPath string) ([]Note, []string, error) {
	var notes []Note
	var directories []string

	// Build absolute path
	targetPath := filepath.Join(s.notesDir, relPath)

	// Read directory contents
	entries, err := os.ReadDir(targetPath)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(targetPath, entry.Name())

		if entry.IsDir() {
			// Skip hidden directories (starting with .)
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}
			directories = append(directories, entry.Name())
		} else if strings.HasSuffix(entry.Name(), ".md") {
			// Get file info for metadata
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Extract metadata from file
			tags, _ := s.extractTags(fullPath)
			keywords, _ := s.extractKeywords(fullPath)
			attendees, _ := s.extractAttendees(fullPath)

			notes = append(notes, Note{
				Name:       entry.Name(),
				FilePath:   fullPath,
				Tags:       tags,
				Keywords:   keywords,
				Attendees:  attendees,
				ModTime:    info.ModTime(),
				IsTemplate: false,
			})
		}
	}

	// Sort directories alphabetically
	sort.Strings(directories)

	// Sort notes by modified time (newest first)
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].ModTime.After(notes[j].ModTime)
	})

	return notes, directories, nil
}

// GetAllDirectories recursively gets all directories under the notes directory
func (s *NotesService) GetAllDirectories() ([]string, error) {
	var directories []string

	err := filepath.Walk(s.notesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if not a directory
		if !info.IsDir() {
			return nil
		}

		// Skip the root notes directory itself
		if path == s.notesDir {
			return nil
		}

		// Skip hidden directories (starting with .)
		if strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}

		// Get relative path from notes directory
		relPath, err := filepath.Rel(s.notesDir, path)
		if err != nil {
			return err
		}

		directories = append(directories, relPath)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort directories alphabetically
	sort.Strings(directories)

	return directories, nil
}
