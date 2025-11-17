package version

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"

	// Repository URL for notetkr
	RepoUrl = "https://github.com/redjax/notetkr"
)

type PackageInfo struct {
	PackageName        string
	RepoUrl            string
	RepoUser           string
	RepoName           string
	PackageVersion     string
	PackageCommit      string
	PackageReleaseDate string
}

// GetPackageInfo returns a struct with information about the current package
func GetPackageInfo() PackageInfo {
	exePath, err := os.Executable()
	binName := "<unknown>"

	if err == nil {
		binName = filepath.Base(exePath)
	}

	repoUser, repoName := parseRepoUrl()

	return PackageInfo{
		PackageName:        binName,
		RepoUrl:            RepoUrl,
		RepoUser:           repoUser,
		RepoName:           repoName,
		PackageVersion:     Version,
		PackageCommit:      Commit,
		PackageReleaseDate: Date,
	}
}

// parseRepoUrl extracts the user/org and repo name from the repository URL
func parseRepoUrl() (user, repo string) {
	u, err := url.Parse(RepoUrl)
	if err != nil {
		return "<unknown>", "<unknown>"
	}

	// Path should be like "/redjax/notetkr"
	path := strings.Trim(u.Path, "/")
	parts := strings.Split(path, "/")

	if len(parts) >= 2 {
		return parts[0], parts[1]
	}

	return "<unknown>", "<unknown>"
}

// GetVersionString returns a formatted version string
func GetVersionString() string {
	return fmt.Sprintf("notetkr version:%s commit:%s date:%s", Version, Commit, Date)
}

// GetShortVersion returns just the version number
func GetShortVersion() string {
	return Version
}
