package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/tui"
	"github.com/spf13/cobra"
)

// NewSearchCmd creates the search command
func NewSearchCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search notes and journals",
		Long:  `Opens the search interface to find content across all notes and journal entries. Optionally provide a search query to start with.`,
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			var query string
			if len(args) > 0 {
				query = args[0]
			}
			runSearch(cfg, query)
		},
	}

	return cmd
}

func runSearch(cfg *config.Config, query string) {
	// Open directly to search view
	app := tui.NewSearchBrowserApp(cfg.JournalDir, cfg.NotesDir, query)

	// Start the TUI
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running search TUI: %v\n", err)
		os.Exit(1)
	}
}
