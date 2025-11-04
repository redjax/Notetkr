package commands

import (
	"fmt"
	"runtime"

	"github.com/redjax/notetkr/internal/config"
	"github.com/spf13/cobra"
)

// NewSelfCmd creates the self command
func NewSelfCmd(getConfig func() *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "self",
		Short: "Manage Notetkr installation",
		Long:  `Commands for managing the Notetkr application itself.`,
	}

	// Add upgrade subcommand
	upgradeCmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade Notetkr to the latest release",
		Long:  `Display instructions for upgrading Notetkr to the latest release.`,
		Run: func(cmd *cobra.Command, args []string) {
			runUpgrade()
		},
	}
	cmd.AddCommand(upgradeCmd)

	return cmd
}

func runUpgrade() {
	fmt.Println("\n╔════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                    Notetkr Upgrade Instructions                           ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════╝\n")

	// Detect platform
	switch runtime.GOOS {
	case "windows":
		fmt.Println("To upgrade Notetkr to the current release, run this command:\n")
		fmt.Println("  & ([scriptblock]::Create((irm https://raw.githubusercontent.com/redjax/Notetkr/refs/heads/main/scripts/install-notetkr.ps1))) -Auto\n")
	case "linux", "darwin":
		fmt.Println("To upgrade Notetkr to the current release, run this command:\n")
		fmt.Println("  curl -LsSf https://raw.githubusercontent.com/redjax/Notetkr/refs/heads/main/scripts/install-notetkr.sh | bash -s -- --auto\n")
	default:
		fmt.Printf("Platform '%s' not recognized. Please visit:\n", runtime.GOOS)
		fmt.Println("  https://github.com/redjax/Notetkr/releases/latest\n")
	}

	fmt.Println("Note: This will download and install the latest release from GitHub.")
	fmt.Println()
}
