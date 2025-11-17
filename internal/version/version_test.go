package version

import (
	"strings"
	"testing"
)

func TestGetPackageInfo(t *testing.T) {
	info := GetPackageInfo()

	// Test that basic fields are populated
	if info.PackageName == "" {
		t.Error("PackageName should not be empty")
	}

	if info.RepoUrl != RepoUrl {
		t.Errorf("RepoUrl mismatch: got %s, want %s", info.RepoUrl, RepoUrl)
	}

	if info.PackageVersion == "" {
		t.Error("PackageVersion should not be empty")
	}

	if info.PackageCommit == "" {
		t.Error("PackageCommit should not be empty")
	}

	if info.PackageReleaseDate == "" {
		t.Error("PackageReleaseDate should not be empty")
	}

	// Verify repo user and name are extracted
	if info.RepoUser == "" || info.RepoUser == "<unknown>" {
		t.Error("RepoUser should be extracted from RepoUrl")
	}

	if info.RepoName == "" || info.RepoName == "<unknown>" {
		t.Error("RepoName should be extracted from RepoUrl")
	}
}

func TestGetVersionString(t *testing.T) {
	versionStr := GetVersionString()

	// Should contain expected components
	if !strings.Contains(versionStr, "notetkr") {
		t.Error("Version string should contain 'notetkr'")
	}

	if !strings.Contains(versionStr, "version:") {
		t.Error("Version string should contain 'version:'")
	}

	if !strings.Contains(versionStr, "commit:") {
		t.Error("Version string should contain 'commit:'")
	}

	if !strings.Contains(versionStr, "date:") {
		t.Error("Version string should contain 'date:'")
	}
}

func TestGetShortVersion(t *testing.T) {
	shortVer := GetShortVersion()

	if shortVer == "" {
		t.Error("Short version should not be empty")
	}

	// Should equal the Version variable
	if shortVer != Version {
		t.Errorf("Short version mismatch: got %s, want %s", shortVer, Version)
	}
}

func TestParseRepoUrl(t *testing.T) {
	user, repo := parseRepoUrl()

	expectedUser := "redjax"
	expectedRepo := "notetkr"

	if user != expectedUser {
		t.Errorf("Repo user mismatch: got %s, want %s", user, expectedUser)
	}

	if repo != expectedRepo {
		t.Errorf("Repo name mismatch: got %s, want %s", repo, expectedRepo)
	}
}
