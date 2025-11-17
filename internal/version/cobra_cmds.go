package version

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewVersionCommand creates a 'version' subcommand that prints the package's version
func NewVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print notetkr's version",
		Long:  "Display version information including git commit and build date.",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(GetVersionString())
		},
	}
}

// NewInfoCommand creates an 'info' subcommand that prints detailed package information
func NewInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show detailed information about notetkr",
		Long:  "Display comprehensive information about the notetkr package including repository details.",
		Run: func(cmd *cobra.Command, args []string) {
			pkgInfo := GetPackageInfo()
			fmt.Printf("Program: %s\n", pkgInfo.PackageName)
			fmt.Printf("Owner: %s\n", pkgInfo.RepoUser)
			fmt.Printf("Repository: %s\n", pkgInfo.RepoName)
			fmt.Printf("Repository URL: %s\n", pkgInfo.RepoUrl)
			fmt.Printf("Version: %s\n", pkgInfo.PackageVersion)
			fmt.Printf("Commit: %s\n", pkgInfo.PackageCommit)
			fmt.Printf("Build Date: %s\n", pkgInfo.PackageReleaseDate)
		},
	}
}
