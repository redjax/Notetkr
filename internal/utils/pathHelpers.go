package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// GetAppDataDir returns the application data directory based on OS
func GetAppDataDir() (string, error) {
	var appDataDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: Use LOCALAPPDATA environment variable
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			// Fallback to APPDATA if LOCALAPPDATA is not set
			appData := os.Getenv("APPDATA")
			if appData == "" {
				return "", fmt.Errorf("neither LOCALAPPDATA nor APPDATA environment variables are set")
			}
			appDataDir = filepath.Join(appData, "notetkr")
		} else {
			appDataDir = filepath.Join(localAppData, "notetkr")
		}
	case "linux":
		// Linux: Follow XDG Base Directory specification
		xdgDataHome := os.Getenv("XDG_DATA_HOME")
		if xdgDataHome == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get user home directory: %w", err)
			}
			xdgDataHome = filepath.Join(homeDir, ".local", "share")
		}
		appDataDir = filepath.Join(xdgDataHome, "notetkr")
	case "darwin":
		// macOS: Use Application Support directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		appDataDir = filepath.Join(homeDir, "Library", "Application Support", "notetkr")
	default:
		// Unknown OS: Use home directory fallback
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		appDataDir = filepath.Join(homeDir, ".notetkr")
	}

	return appDataDir, nil
}

// EnsureAppDataDirs creates the necessary application data directories
func EnsureAppDataDirs() error {
	dirs := []func() (string, error){
		GetAppDataDir,
	}

	for _, dirFunc := range dirs {
		dir, err := dirFunc()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
