package commands

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/redjax/notetkr/internal/config"
	"github.com/spf13/cobra"
)

// NewImportCmd creates the import command
func NewImportCmd(getConfig func() *config.Config) *cobra.Command {
	var filePath string

	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import notes and journals from a ZIP archive",
		Long: `Import notes and journals from a ZIP archive created by the export command.
Merges with existing data, keeping the newer version of any duplicate files.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			if filePath == "" {
				fmt.Fprintf(os.Stderr, "Error: -f/--file flag is required\n")
				cmd.Usage()
				os.Exit(1)
			}
			if err := runImport(cfg, filePath); err != nil {
				fmt.Fprintf(os.Stderr, "Error importing data: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&filePath, "file", "f", "", "Path to the ZIP file to import (required)")
	cmd.MarkFlagRequired("file")

	return cmd
}

func runImport(cfg *config.Config, zipPath string) error {
	// Open ZIP file
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	filesImported := 0
	filesSkipped := 0
	filesUpdated := 0

	// Process each file in the ZIP
	for _, file := range reader.File {
		// Skip directories
		if file.FileInfo().IsDir() {
			continue
		}

		// Determine destination path
		var destPath string
		zipPath := filepath.ToSlash(file.Name)

		if strings.HasPrefix(zipPath, "notes/") {
			// Extract to notes directory
			relPath := strings.TrimPrefix(zipPath, "notes/")
			destPath = filepath.Join(cfg.NotesDir, relPath)
		} else if strings.HasPrefix(zipPath, "journals/") {
			// Extract to journals directory
			relPath := strings.TrimPrefix(zipPath, "journals/")
			destPath = filepath.Join(cfg.JournalDir, relPath)
		} else {
			// Unknown path, skip
			continue
		}

		// Check if file exists and compare modification times
		shouldExtract := true

		if stat, err := os.Stat(destPath); err == nil {
			// File exists, compare modification times
			existingModTime := stat.ModTime()
			zipModTime := file.Modified

			if zipModTime.Before(existingModTime) || zipModTime.Equal(existingModTime) {
				// Existing file is newer or same age, skip
				shouldExtract = false
				filesSkipped++
			} else {
				filesUpdated++
			}
		} else {
			filesImported++
		}

		if shouldExtract {
			if err := extractFile(file, destPath); err != nil {
				return fmt.Errorf("failed to extract %s: %w", file.Name, err)
			}
		}
	}

	fmt.Printf("✓ Import complete:\n")
	fmt.Printf("  - %d new file(s) imported\n", filesImported)
	fmt.Printf("  - %d file(s) updated (newer version)\n", filesUpdated)
	fmt.Printf("  - %d file(s) skipped (existing version is newer)\n", filesSkipped)

	return nil
}

// extractFile extracts a single file from the ZIP archive
func extractFile(zipFile *zip.File, destPath string) error {
	// Create destination directory if it doesn't exist
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Open source file from ZIP
	srcFile, err := zipFile.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in ZIP: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy contents
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	// Set modification time to match the ZIP entry
	if err := os.Chtimes(destPath, zipFile.Modified, zipFile.Modified); err != nil {
		// Non-fatal error, just log it
		fmt.Fprintf(os.Stderr, "Warning: failed to set modification time for %s: %v\n", destPath, err)
	}

	return nil
}
