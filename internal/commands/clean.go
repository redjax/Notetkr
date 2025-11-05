package commands

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/services"
	"github.com/redjax/notetkr/internal/tui"
	"github.com/spf13/cobra"
)

// NewCleanCmd creates the clean command
func NewCleanCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean up and optimize your notes",
		Long:  `Opens a menu with options for automated cleanup tasks like removing unused images and deduplicating content.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runCleanMenu(cfg)
		},
	}

	// Add images subcommand
	imagesCmd := &cobra.Command{
		Use:   "images",
		Short: "Clean up image attachments",
		Long:  `Removes unused images and deduplicates image files across notes and journals.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runCleanImages(cfg)
		},
	}
	cmd.AddCommand(imagesCmd)

	// Add notes subcommand
	notesCmd := &cobra.Command{
		Use:   "notes",
		Short: "Clean up empty notes",
		Long:  `Removes notes that only contain the default template with no user content.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runCleanNotes(cfg)
		},
	}
	cmd.AddCommand(notesCmd)

	// Add journals subcommand
	journalsCmd := &cobra.Command{
		Use:   "journals",
		Short: "Clean up empty journal entries",
		Long:  `Removes journal entries that only contain the default template with no user content.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runCleanJournals(cfg)
		},
	}
	cmd.AddCommand(journalsCmd)

	return cmd
}

func runCleanMenu(cfg *config.Config) {
	// Launch the clean menu TUI
	app := tui.NewCleanMenuApp(cfg)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running clean menu: %v\n", err)
	}
}

func runCleanImages(cfg *config.Config) {
	// Create cleanup service
	cleanupService := services.NewCleanupService(cfg.NotesDir, cfg.JournalDir)

	// Run the cleanup with a progress TUI
	app := tui.NewCleanImagesApp(cleanupService)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running image cleanup: %v\n", err)
	}
}

func runCleanNotes(cfg *config.Config) {
	cleanupService := services.NewCleanupService(cfg.NotesDir, cfg.JournalDir)

	fmt.Println("🔍 Scanning for empty notes...")
	deleted, err := cleanupService.CleanEmptyNotes()
	if err != nil {
		fmt.Printf("❌ Error cleaning notes: %v\n", err)
		return
	}

	if deleted == 0 {
		fmt.Println("✓ No empty notes found")
	} else {
		fmt.Printf("✓ Deleted %d empty note(s)\n", deleted)
	}
}

func runCleanJournals(cfg *config.Config) {
	cleanupService := services.NewCleanupService(cfg.NotesDir, cfg.JournalDir)

	fmt.Println("🔍 Scanning for empty journal entries...")
	deleted, err := cleanupService.CleanEmptyJournals()
	if err != nil {
		fmt.Printf("❌ Error cleaning journals: %v\n", err)
		return
	}

	if deleted == 0 {
		fmt.Println("✓ No empty journal entries found")
	} else {
		fmt.Printf("✓ Deleted %d empty journal entr(ies)\n", deleted)
	}
}
