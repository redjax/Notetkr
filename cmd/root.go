package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/redjax/notetkr/internal/commands"
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/tui"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	debug   bool
	k       = koanf.New(".")
	cfg     *config.Config
)

var rootCmd = &cobra.Command{
	Use:   `nt`,
	Short: `Notetkr is a terminal-based note-taking and journaling app that uses Markdown for the notes.`,
	// Long: ``
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, launch the dashboard TUI
		runDashboard()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config-file", "c", "", "config file (supports .yml, .json, .toml, .env)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "Enable debug logging")

	// Add subcommands - they will get config when executed
	rootCmd.AddCommand(commands.NewJournalCmd(func() *config.Config { return cfg }))
	rootCmd.AddCommand(commands.NewNotesCmd(func() *config.Config { return cfg }))
	rootCmd.AddCommand(commands.NewSearchCmd(func() *config.Config { return cfg }))
	rootCmd.AddCommand(commands.NewExportCmd(func() *config.Config { return cfg }))
	rootCmd.AddCommand(commands.NewImportCmd(func() *config.Config { return cfg }))

	// Handle persistent flags
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		// Handle --debug flag
		if d, _ := cmd.Flags().GetBool("debug"); d {
			log.SetFlags(log.LstdFlags | log.Lshortfile)
			log.Println("DEBUG logging enabled")
		}
	}

	cobra.OnInitialize(initConfig)
}

func initConfig() {
	// Initialize config with defaults
	cfg = config.DefaultConfig()

	if cfgFile != "" {
		if err := k.Load(file.Provider(cfgFile), json.Parser()); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config file: %v\n", err)
			os.Exit(1)
		}
	}

	// Load from environment variables
	k.Load(env.Provider("NOTETKR_", ".", func(s string) string {
		return strings.Replace(strings.ToLower(
			strings.TrimPrefix(s, "NOTETKR_")), "_", ".", -1)
	}), nil)

	// Unmarshal config values into the struct
	if err := k.Unmarshal("", cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling config: %v\n", err)
		os.Exit(1)
	}

	// Ensure data directories exist
	ensureDataDirs()
}

func ensureDataDirs() {
	dirs := []string{
		cfg.DataDir,
		cfg.NotesDir,
		cfg.JournalDir,
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating directory %s: %v\n", dir, err)
			os.Exit(1)
		}
	}
}

func runDashboard() {
	// Ensure directories exist
	ensureDataDirs()

	// Create and run the dashboard TUI
	app := tui.NewAppModel(cfg.JournalDir, cfg.NotesDir)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running dashboard: %v\n", err)
		os.Exit(1)
	}
}
