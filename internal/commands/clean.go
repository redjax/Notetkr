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
