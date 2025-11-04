package services

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CleanupService handles cleanup operations for notes and journals
type CleanupService struct {
	notesDir   string
	journalDir string
}

// CleanupStats tracks cleanup statistics
type CleanupStats struct {
	UnusedImagesDeleted    int
	DuplicateImagesDeleted int
	ReferencesUpdated      int
	BytesFreed             int64
}

// ImageReference tracks where an image is referenced
type ImageReference struct {
	FilePath  string
	LineNum   int
	ImagePath string
}

func NewCleanupService(notesDir, journalDir string) *CleanupService {
	return &CleanupService{
		notesDir:   notesDir,
		journalDir: journalDir,
	}
}

// CleanImages performs a complete image cleanup:
// 1. Removes images not referenced in any notes/journals
// 2. Deduplicates images by content hash
// 3. Updates references to point to deduplicated images
func (s *CleanupService) CleanImages() (*CleanupStats, error) {
	stats := &CleanupStats{}

	// Step 1: Find all image files
	imageFiles, err := s.findAllImages()
	if err != nil {
		return stats, fmt.Errorf("failed to find images: %w", err)
	}

	// Step 2: Find all image references in markdown files
	references, err := s.findAllImageReferences()
	if err != nil {
		return stats, fmt.Errorf("failed to find image references: %w", err)
	}

	// Step 3: Build a map of referenced images
	referencedImages := make(map[string]bool)
	for _, ref := range references {
		// Normalize the path
		normalizedPath := s.normalizeImagePath(ref.ImagePath, filepath.Dir(ref.FilePath))
		referencedImages[normalizedPath] = true
	}

	// Step 4: Delete unreferenced images
	for _, imgPath := range imageFiles {
		if !referencedImages[imgPath] {
			info, err := os.Stat(imgPath)
			if err == nil {
				stats.BytesFreed += info.Size()
			}

			if err := os.Remove(imgPath); err == nil {
				stats.UnusedImagesDeleted++
			}
		}
	}

	// Step 5: Re-scan for remaining images and deduplicate
	remainingImages, err := s.findAllImages()
	if err != nil {
		return stats, fmt.Errorf("failed to re-scan images: %w", err)
	}

	// Build hash map for deduplication
	hashToPath := make(map[string]string)
	duplicates := make(map[string]string) // duplicate path -> canonical path

	for _, imgPath := range remainingImages {
		hash, err := s.hashFile(imgPath)
		if err != nil {
			continue
		}

		if canonicalPath, exists := hashToPath[hash]; exists {
			// Found a duplicate
			duplicates[imgPath] = canonicalPath
		} else {
			// First occurrence of this hash
			hashToPath[hash] = imgPath
		}
	}

	// Step 6: Update references to duplicates and delete duplicate files
	for duplicatePath, canonicalPath := range duplicates {
		// Update all references to the duplicate
		updated, err := s.updateImageReferences(duplicatePath, canonicalPath)
		if err != nil {
			continue
		}
		stats.ReferencesUpdated += updated

		// Delete the duplicate file
		info, err := os.Stat(duplicatePath)
		if err == nil {
			stats.BytesFreed += info.Size()
		}

		if err := os.Remove(duplicatePath); err == nil {
			stats.DuplicateImagesDeleted++
		}
	}

	return stats, nil
}

// findAllImages finds all image files in .attachments directories
func (s *CleanupService) findAllImages() ([]string, error) {
	var images []string
	imageExts := map[string]bool{
		".png": true, ".jpg": true, ".jpeg": true,
		".gif": true, ".bmp": true, ".webp": true,
	}

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if info.IsDir() {
			return nil
		}

		// Check if file is in an .attachments directory
		if strings.Contains(path, ".attachments") {
			ext := strings.ToLower(filepath.Ext(path))
			if imageExts[ext] {
				images = append(images, path)
			}
		}

		return nil
	}

	// Walk both notes and journal directories
	if err := filepath.Walk(s.notesDir, walkFunc); err != nil {
		return nil, err
	}
	if err := filepath.Walk(s.journalDir, walkFunc); err != nil {
		return nil, err
	}

	return images, nil
}

// findAllImageReferences finds all markdown image references
func (s *CleanupService) findAllImageReferences() ([]ImageReference, error) {
	var references []ImageReference

	// Regex to match markdown image syntax: ![alt](.attachments/path/to/image.png)
	imageRegex := regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() || !strings.HasSuffix(info.Name(), ".md") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			matches := imageRegex.FindAllStringSubmatch(line, -1)
			for _, match := range matches {
				if len(match) > 2 {
					imagePath := match[2]
					// Only track .attachments images
					if strings.Contains(imagePath, ".attachments") {
						references = append(references, ImageReference{
							FilePath:  path,
							LineNum:   lineNum,
							ImagePath: imagePath,
						})
					}
				}
			}
		}

		return nil
	}

	if err := filepath.Walk(s.notesDir, walkFunc); err != nil {
		return nil, err
	}
	if err := filepath.Walk(s.journalDir, walkFunc); err != nil {
		return nil, err
	}

	return references, nil
}

// normalizeImagePath converts a relative image path to an absolute path
func (s *CleanupService) normalizeImagePath(imagePath, baseDir string) string {
	// If already absolute, return as-is
	if filepath.IsAbs(imagePath) {
		return filepath.Clean(imagePath)
	}

	// Join with base directory and clean
	absPath := filepath.Join(baseDir, imagePath)
	return filepath.Clean(absPath)
}

// hashFile computes SHA256 hash of a file
func (s *CleanupService) hashFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// updateImageReferences updates all references from oldPath to newPath
func (s *CleanupService) updateImageReferences(oldPath, newPath string) (int, error) {
	updated := 0

	// Find all references to the old path
	references, err := s.findAllImageReferences()
	if err != nil {
		return 0, err
	}

	// Group references by file
	fileRefs := make(map[string][]ImageReference)
	for _, ref := range references {
		normalizedRef := s.normalizeImagePath(ref.ImagePath, filepath.Dir(ref.FilePath))
		if normalizedRef == oldPath {
			fileRefs[ref.FilePath] = append(fileRefs[ref.FilePath], ref)
		}
	}

	// Update each file
	for filePath, refs := range fileRefs {
		if err := s.updateFileReferences(filePath, refs, oldPath, newPath); err != nil {
			continue
		}
		updated += len(refs)
	}

	return updated, nil
}

// updateFileReferences updates image references in a single file
func (s *CleanupService) updateFileReferences(filePath string, refs []ImageReference, oldPath, newPath string) error {
	// Read the entire file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Calculate the relative path from this file to the new image
	fileDir := filepath.Dir(filePath)
	relPath, err := filepath.Rel(fileDir, newPath)
	if err != nil {
		return err
	}

	// Convert to forward slashes for markdown
	relPath = filepath.ToSlash(relPath)

	// Replace all occurrences
	contentStr := string(content)
	for _, ref := range refs {
		// Create a regex to match this specific image reference
		escapedPath := regexp.QuoteMeta(ref.ImagePath)
		pattern := regexp.MustCompile(fmt.Sprintf(`!\[([^\]]*)\]\(%s\)`, escapedPath))
		contentStr = pattern.ReplaceAllString(contentStr, fmt.Sprintf(`![$1](%s)`, relPath))
	}

	// Write back to file
	return os.WriteFile(filePath, []byte(contentStr), 0644)
}
