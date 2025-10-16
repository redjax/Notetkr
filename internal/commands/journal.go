package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/services"
	"github.com/redjax/notetkr/internal/tui"
	"github.com/spf13/cobra"
)

// NewJournalCmd creates the journal command
func NewJournalCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Open today's journal entry",
		Long:  `Opens today's journal entry in a TUI. The journal will be created in the configured journal directory.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runJournal(cfg)
		},
	}

	return cmd
}

func runJournal(cfg *config.Config) {
	// Ensure journal directory exists
	journalService := services.NewJournalService(cfg.JournalDir)

	// Create journal model
	journalModel := tui.NewJournalModel(journalService)

	// Start the TUI
	p := tea.NewProgram(journalModel, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running journal TUI: %v\n", err)
		os.Exit(1)
	}
}
