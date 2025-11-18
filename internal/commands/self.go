package commands

import (
	"github.com/redjax/notetkr/internal/config"
	"github.com/redjax/notetkr/internal/version"
	"github.com/spf13/cobra"
)

// NewSelfCmd creates the self command
func NewSelfCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self",
		Short: "Manage Notetkr installation",
		Long:  `Commands for managing the Notetkr application itself.`,
	}

	// Add version subcommand
	cmd.AddCommand(version.NewVersionCommand())

	// Add info subcommand
	cmd.AddCommand(version.NewInfoCommand())

	// Add upgrade subcommand
	cmd.AddCommand(version.NewUpgradeCommand())

	return cmd
}
