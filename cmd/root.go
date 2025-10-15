package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"

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
)

var rootCmd = &cobra.Command{
	Use:   `nt`,
	Short: `Notetkr is a terminal-based note-taking and journaling app that uses Markdown for the notes.`,
	// Long: ``
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (JSON)")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, "Enable debug logging")

	// Add subcommands
	// rootCmd.AddCommand(someCmd)

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

	// Optionally, unmarshal config object into a struct
	// k.Unmarshal("", &yourConfigStruct)
}
