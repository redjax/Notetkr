package main

import (
	// "github.com/redjax/notetkr/internal/version"

	// Import the cmd directory with root.go
	"github.com/redjax/notetkr/cmd"
)

func main() {
	// Check if an update is needed
	// version.TrySelfUpgrade()

	// Call the root command
	cmd.Execute()
}
