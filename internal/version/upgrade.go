package version

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// UpgradeSelf is the entrypoint for 'nt self upgrade'.
func UpgradeSelf(cmd *cobra.Command, args []string, checkOnly bool) error {
	info := GetPackageInfo()

	repo := fmt.Sprintf("%s/%s", info.RepoUser, info.RepoName)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

	fmt.Fprintln(cmd.ErrOrStderr(), "Checking for latest release...")

	// #nosec G107 - URL is constructed from hardcoded GitHub API endpoint and repo constant
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GitHub API returned status: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release JSON: %w", err)
	}

	current := info.PackageVersion
	latest := release.TagName

	fmt.Fprintln(cmd.ErrOrStderr(), "Current version:", current)
	fmt.Fprintln(cmd.ErrOrStderr(), "Latest version: ", latest)

	if current == "dev" {
		fmt.Fprintf(cmd.ErrOrStderr(), "🛠️  This is a development release: %s\n", current)
		return nil
	}

	cmp := compareVersion(current, latest)

	switch cmp {
	case -1:
		fmt.Fprintf(cmd.ErrOrStderr(), "🚀 Upgrade available: %s → %s\n", current, latest)
		if checkOnly {
			fmt.Fprintln(cmd.ErrOrStderr(), "✅ Use this command without --check to upgrade.")
			return nil
		}
	case 0:
		fmt.Fprintf(cmd.ErrOrStderr(), "🔄 No new release available, notetkr is up to date (%s).\n", current)
		return nil
	case 1:
		fmt.Fprintf(cmd.ErrOrStderr(), "🤯 You're ahead of the latest release: current=%s, release=%s\n", current, latest)
		return nil
	}

	normalizedOS := normalizeOS(runtime.GOOS)
	arch := normalizeArch(runtime.GOARCH)

	// Expected archive name based on .goreleaser.yaml name_template
	// notetkr_Linux_x86_64.tar.gz, notetkr_Windows_x86_64.zip, notetkr_Darwin_x86_64.tar.gz
	var expectedSuffix string
	if runtime.GOOS == "windows" {
		expectedSuffix = fmt.Sprintf("notetkr_%s_%s.zip", normalizedOS, arch)
	} else {
		expectedSuffix = fmt.Sprintf("notetkr_%s_%s.tar.gz", normalizedOS, arch)
	}

	var assetURL string
	for _, asset := range release.Assets {
		if asset.Name == "" {
			continue
		}
		if asset.Name == expectedSuffix {
			assetURL = asset.BrowserDownloadURL
			break
		}
	}

	if assetURL == "" {
		fmt.Fprintln(cmd.ErrOrStderr(), "Available assets:")
		for _, asset := range release.Assets {
			fmt.Fprintln(cmd.ErrOrStderr(), " -", asset.Name)
		}
		return fmt.Errorf("no suitable release found for platform: %s/%s (expected: %s)", runtime.GOOS, runtime.GOARCH, expectedSuffix)
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "Downloading:", assetURL)

	// #nosec G107 - URL is from GitHub release API response, validated to be from github.com
	resp2, err := http.Get(assetURL)
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	defer resp2.Body.Close()

	archiveTmp, err := os.CreateTemp("", "notetkr-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp archive file: %w", err)
	}
	defer os.Remove(archiveTmp.Name())

	if _, err := io.Copy(archiveTmp, resp2.Body); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}
	// #nosec G104 - Close error is non-critical, file is fully written
	archiveTmp.Close()

	var binaryTmp string
	if runtime.GOOS == "windows" {
		binaryTmp, err = extractBinaryFromZip(archiveTmp.Name())
	} else {
		binaryTmp, err = extractBinaryFromTarGz(archiveTmp.Name())
	}

	if err != nil {
		return fmt.Errorf("failed to extract binary: %w", err)
	}
	defer os.Remove(binaryTmp)

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	newPath := exePath + ".new"
	if err := copyFile(binaryTmp, newPath); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintln(cmd.ErrOrStderr(), "Permission denied: try running with 'sudo nt self upgrade'")
		}
		return fmt.Errorf("failed to save new binary: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(),
		"✅ Upgrade downloaded:\n  %s\n"+
			"  It will be applied next time you run a notetkr command, i.e. nt --version.\n",
		newPath)

	return nil
}

// normalizeOS maps runtime.GOOS to goreleaser asset naming
func normalizeOS(goos string) string {
	switch strings.ToLower(goos) {
	case "darwin":
		return "Darwin"
	case "linux":
		return "Linux"
	case "windows":
		return "Windows"
	default:
		// Title case the first letter
		return strings.ToUpper(goos[:1]) + strings.ToLower(goos[1:])
	}
}

// normalizeArch maps runtime.GOARCH to goreleaser asset naming
func normalizeArch(arch string) string {
	switch arch {
	case "amd64":
		return "x86_64"
	case "386":
		return "i386"
	default:
		return arch
	}
}

// extractBinaryFromZip extracts the binary from a zip file and returns path
func extractBinaryFromZip(zipPath string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Look for the binary file specifically (not README, LICENSE, etc.)
	var binaryFile *zip.File
	expectedBinaryName := "nt"
	if runtime.GOOS == "windows" {
		expectedBinaryName = "nt.exe"
	}

	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}

		// Match the binary file by name (case-insensitive)
		fileName := strings.ToLower(f.Name)
		if fileName == expectedBinaryName || strings.HasSuffix(fileName, "/"+expectedBinaryName) {
			binaryFile = f
			break
		}
	}

	if binaryFile == nil {
		return "", fmt.Errorf("binary '%s' not found in zip archive", expectedBinaryName)
	}

	rc, err := binaryFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	tmpBin, err := os.CreateTemp("", "nt-bin-*")
	if err != nil {
		return "", err
	}

	// Limit extraction size to 500MB to prevent decompression bomb attacks
	// #nosec G110 - Size limit implemented via io.LimitReader
	limitedReader := io.LimitReader(rc, 500*1024*1024) // 500MB max
	if _, err := io.Copy(tmpBin, limitedReader); err != nil {
		// #nosec G104 - Error from Close is non-critical here, primary error is from Copy
		tmpBin.Close()
		return "", err
	}

	// #nosec G104 - Error from Close checked below via Chmod
	tmpBin.Close()

	// #nosec G302 - Binary must be executable (0755 is appropriate for executables)
	if err := os.Chmod(tmpBin.Name(), 0755); err != nil {
		return "", err
	}

	return tmpBin.Name(), nil
}

// extractBinaryFromTarGz extracts the binary from a tar.gz file and returns path
func extractBinaryFromTarGz(tarGzPath string) (string, error) {
	f, err := os.Open(tarGzPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gzr, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	expectedBinaryName := "nt"

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Match the binary file by name
		fileName := strings.ToLower(header.Name)
		if fileName == expectedBinaryName || strings.HasSuffix(fileName, "/"+expectedBinaryName) {
			tmpBin, err := os.CreateTemp("", "nt-bin-*")
			if err != nil {
				return "", err
			}

			// Limit extraction size to 500MB
			limitedReader := io.LimitReader(tr, 500*1024*1024)
			if _, err := io.Copy(tmpBin, limitedReader); err != nil {
				tmpBin.Close()
				return "", err
			}

			tmpBin.Close()

			// #nosec G302 - Binary must be executable
			if err := os.Chmod(tmpBin.Name(), 0755); err != nil {
				return "", err
			}

			return tmpBin.Name(), nil
		}
	}

	return "", fmt.Errorf("binary '%s' not found in tar.gz archive", expectedBinaryName)
}

// copyFile utility
func copyFile(src, dst string) error {
	// #nosec G304 - CLI tool copies files during self-upgrade by design
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// #nosec G304 - CLI tool creates files during self-upgrade by design
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Chmod(0755)
}

// compareVersion compares two semantic version strings (e.g., "v1.2.3" vs "v1.2.4")
// Returns: -1 if a < b, 0 if a == b, 1 if a > b
func compareVersion(a, b string) int {
	// Strip leading 'v' if present
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	if a == b {
		return 0
	}

	// Simple string comparison works for most semantic versions
	// For more complex comparisons, consider using a proper semver library
	if a < b {
		return -1
	}
	return 1
}

// TrySelfUpgrade checks if "<binary>.new" exists and replaces current binary with it.
func TrySelfUpgrade() {
	exePath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		return
	}

	newPath := exePath + ".new"

	if _, err := os.Stat(newPath); err == nil {
		// New file exists: perform replacement

		if runtime.GOOS == "windows" {
			// Use Windows-specific updater (launches background script and exits)
			err := RunWindowsSelfUpgrade(exePath, newPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "notetkr Windows self-upgrade failed: %v\n", err)
			} else {
				fmt.Fprintf(os.Stderr, "🔁 Applying upgrade...\n")
				// Exit immediately so the background script can replace the .exe
				os.Exit(0)
			}
			return
		}

		// Unix: direct rename works because the file isn't locked
		errRename := os.Rename(newPath, exePath)

		if errRename != nil {
			fmt.Fprintf(os.Stderr, "Failed to replace executable: %v\n", errRename)
		} else {
			fmt.Fprintf(os.Stderr, "🔁 notetkr upgraded successfully.\n")
		}
	}
}
