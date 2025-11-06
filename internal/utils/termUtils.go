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

// CenterContent takes a string and centers it horizontally with padding
// It calculates the appropriate left padding to center the content
func CenterContent(content string, termWidth int) string {
	if termWidth <= 0 {
		return content
	}

	// Split content into lines to find the longest line
	lines := splitLines(content)
	maxLineLen := 0
	for _, line := range lines {
		// Strip ANSI codes for accurate length calculation
		visibleLen := visibleLength(line)
		if visibleLen > maxLineLen {
			maxLineLen = visibleLen
		}
	}

	// Calculate left padding to center the content
	leftPadding := (termWidth - maxLineLen) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	// Add padding to each line
	paddedLines := make([]string, len(lines))
	padding := ""
	for i := 0; i < leftPadding; i++ {
		padding += " "
	}

	for i, line := range lines {
		paddedLines[i] = padding + line
	}

	return joinLines(paddedLines)
}

func splitLines(s string) []string {
	lines := []string{}
	current := ""
	for _, ch := range s {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

func joinLines(lines []string) string {
	result := ""
	for i, line := range lines {
		result += line
		if i < len(lines)-1 {
			result += "\n"
		}
	}
	return result
}

// visibleLength calculates the visible length of a string, ignoring ANSI escape codes
func visibleLength(s string) int {
	length := 0
	inEscape := false

	for i := 0; i < len(s); i++ {
		if s[i] == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		length++
	}

	return length
}
