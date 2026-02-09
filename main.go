package main

import (
	"errors"
	"os"

	"github.com/ishuar/tfskel/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// Check if it's an ExitError with a specific code
		var exitErr *cmd.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
