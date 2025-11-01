package commands

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/tui"
	"github.com/spf13/cobra"
)

// NewNotesCmd creates the notes command
func NewNotesCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "notes",
		Short: "Browse and manage notes",
		Long:  `Open the notes browser to view, search, create, and manage your notes.`,
		Run: func(cmd *cobra.Command, args []string) {
			cfg := getConfig()
			runNotes(cfg)
		},
	}

	return cmd
}

func runNotes(cfg *config.Config) {
	// Open directly to notes view
	app := tui.NewNotesBrowserApp(cfg.JournalDir, cfg.NotesDir)

	// Start the TUI
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running notes TUI: %v\n", err)
		os.Exit(1)
	}
}
