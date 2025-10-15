package utils

import (
	"time"

	"github.com/briandowns/spinner"
)

// SpinnerUtil provides methods to show progress indicators
type SpinnerUtil struct {
	s *spinner.Spinner
}

// NewSpinnerService creates a new spinner service
func NewSpinnerService() *SpinnerUtil {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	return &SpinnerUtil{s: s}
}

// Start begins the spinner with the given message
func (s *SpinnerUtil) Start(message string) {
	s.s.Suffix = " " + message
	s.s.Start()
}

// Stop stops the spinner
func (s *SpinnerUtil) Stop() {
	s.s.Stop()
}

// Success stops the spinner and displays a success message
func (s *SpinnerUtil) Success(message string) {
	s.s.FinalMSG = "✓ " + message + "\n"
	s.s.Stop()
}

// Error stops the spinner and displays an error message
func (s *SpinnerUtil) Error(message string) {
	s.s.FinalMSG = "✗ " + message + "\n"
	s.s.Stop()
}
