package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/tui"
	"github.com/spf13/cobra"
)

// NewJournalCmd creates the journal command
func NewJournalCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "journal",
		Short: "Browse and manage journal entries",
		Long:  `Opens the journal browser to view and edit journal entries organized by date.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runJournal(cfg)
		},
	}

	return cmd
}

func runJournal(cfg *config.Config) {
	// Open directly to journals view
	app := tui.NewJournalBrowserApp(cfg.JournalDir, cfg.NotesDir)

	// Start the TUI
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running journal TUI: %v\n", err)
		os.Exit(1)
	}
}
