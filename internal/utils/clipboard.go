package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CopyToClipboard copies the given text to the system clipboard
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("clip")
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel as fallback
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (install xclip or xsel)")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	if cmd == nil {
		return fmt.Errorf("failed to create clipboard command for %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("clipboard command failed: %v", err)
	}
	return nil
}

// isRunningInWSL detects if we're running in Windows Subsystem for Linux
func isRunningInWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check for WSL environment variables
	if os.Getenv("WSL_DISTRO_NAME") != "" || os.Getenv("WSLENV") != "" {
		return true
	}

	// Check /proc/version for Microsoft signature
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := strings.ToLower(string(data))
		if strings.Contains(version, "microsoft") || strings.Contains(version, "wsl") {
			return true
		}
	}

	return false
}

// PromptUserChoice displays a numbered list and prompts user to select one
func PromptUserChoice(items []string, itemType string) (int, error) {
	if len(items) == 0 {
		return -1, fmt.Errorf("no items to choose from")
	}

	if len(items) == 1 {
		fmt.Printf("Only one %s found. Copying to clipboard...\n", itemType)
		return 0, nil
	}

	fmt.Printf("\nFound %d %s(s). Select one to copy to clipboard:\n\n", len(items), itemType)

	for i, item := range items {
		fmt.Printf("%d. %s\n", i+1, item)
	}

	fmt.Print("\nEnter number (1-" + strconv.Itoa(len(items)) + "): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return -1, fmt.Errorf("failed to read input: %v", err)
	}

	input = strings.TrimSpace(input)
	choice, err := strconv.Atoi(input)
	if err != nil {
		return -1, fmt.Errorf("invalid input: please enter a number")
	}

	if choice < 1 || choice > len(items) {
		return -1, fmt.Errorf("invalid choice: must be between 1 and %d", len(items))
	}

	// Convert from 1-based user input to 0-based array index
	return choice - 1, nil
}
