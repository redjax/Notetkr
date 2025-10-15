package utils

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"golang.org/x/term"
)

// DetectTerminalWidth tries to get the terminal width, falling back to a default if necessary.
func DetectTerminalWidth(fallback int) int {
	fd := os.Stdout.Fd()
	if isatty.IsTerminal(fd) {
		w, _, err := term.GetSize(int(fd))
		if err == nil && w >= 80 {
			return w
		}
	}
	return fallback
}

// MaxNameLen calculates the max length for the "Name" column given terminal width and other column widths.
func MaxNameLen(termWidth, idCol, valueCol, borders int) int {
	maxNameLen := termWidth - (idCol + valueCol + borders)
	if maxNameLen < 10 {
		maxNameLen = 10
	}
	return maxNameLen
}

// SafeStringDeref safely dereferences a string pointer, returning empty string if nil
func SafeStringDeref(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

// DisplayWithPager displays content using a simple pager or directly to stdout
func DisplayWithPager(content, title string) {
	// For now, just print directly. Later can be enhanced with actual pager
	fmt.Printf("=== %s ===\n", title)
	fmt.Print(content)
}
