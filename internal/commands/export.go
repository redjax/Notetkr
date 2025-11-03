package commands

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/redjax/notetkr/internal/config"
	"github.com/spf13/cobra"
)

// NewExportCmd creates the export command
func NewExportCmd(getConfig func() *config.Config) *cobra.Command {
	var outputPath string
	var exportTypes []string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export notes and journals to a ZIP archive",
		Long: `Export your notes and journals to a ZIP archive. By default, exports the entire data directory.
Use -t/--export-type to specify what to export (notes, journals, or both).`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			if err := runExport(cfg, outputPath, exportTypes); err != nil {
				fmt.Fprintf(os.Stderr, "Error exporting data: %v\n", err)
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Output path for the ZIP file")
	cmd.Flags().StringSliceVarP(&exportTypes, "export-type", "t", []string{}, "What to export: notes, journals (default: both)")

	return cmd
}

func runExport(cfg *config.Config, outputPath string, exportTypes []string) error {
	// Determine output path
	if outputPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		timestamp := time.Now().Format("2006-01-02")
		outputPath = filepath.Join(homeDir, fmt.Sprintf("%s-notetkr-data.zip", timestamp))
	} else {
		// Ensure output path ends with .zip
		if !strings.HasSuffix(strings.ToLower(outputPath), ".zip") {
			outputPath += ".zip"
		}
	}

	// Determine what to export
	exportNotes := true
	exportJournals := true

	if len(exportTypes) > 0 {
		exportNotes = false
		exportJournals = false

		for _, t := range exportTypes {
			switch strings.ToLower(t) {
			case "notes":
				exportNotes = true
			case "journals":
				exportJournals = true
			default:
				return fmt.Errorf("invalid export type: %s (valid options: notes, journals)", t)
			}
		}
	}

	// Create ZIP file
	zipFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create ZIP file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Track files added
	filesAdded := 0

	// Export notes if requested
	if exportNotes {
		count, err := addDirToZip(zipWriter, cfg.NotesDir, "notes")
		if err != nil {
			return fmt.Errorf("failed to add notes to archive: %w", err)
		}
		filesAdded += count
	}

	// Export journals if requested
	if exportJournals {
		count, err := addDirToZip(zipWriter, cfg.JournalDir, "journals")
		if err != nil {
			return fmt.Errorf("failed to add journals to archive: %w", err)
		}
		filesAdded += count
	}

	// Close the ZIP writer to flush everything
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to finalize ZIP file: %w", err)
	}

	fmt.Printf("✓ Successfully exported %d file(s) to: %s\n", filesAdded, outputPath)

	return nil
}

// addDirToZip adds all files from a directory to the ZIP archive
func addDirToZip(zipWriter *zip.Writer, sourceDir, basePath string) (int, error) {
	filesAdded := 0

	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access
			return nil
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		// Create ZIP path (use forward slashes for ZIP standard)
		zipPath := filepath.Join(basePath, relPath)
		zipPath = filepath.ToSlash(zipPath)

		// Create file in ZIP
		writer, err := zipWriter.Create(zipPath)
		if err != nil {
			return fmt.Errorf("failed to create file in ZIP: %w", err)
		}

		// Open source file
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open source file: %w", err)
		}
		defer file.Close()

		// Copy file contents to ZIP
		_, err = io.Copy(writer, file)
		if err != nil {
			return fmt.Errorf("failed to write file to ZIP: %w", err)
		}

		filesAdded++
		return nil
	})

	return filesAdded, err
}
