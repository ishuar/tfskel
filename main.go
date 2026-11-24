// Command tfskel is the entry point. It runs the cobra tree built in
// package cmd and maps a returned *cmd.ExitError to the process exit code,
// so validate (2) and review plan (2, 3) can signal findings to CI without
// going through the generic error path (1).
package main

import (
	"errors"
	"os"

	"github.com/ishuar/tfskel/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// Check if it's an ExitError with a specific code
		if exitErr, ok := errors.AsType[*cmd.ExitError](err); ok {
			os.Exit(exitErr.Code)
		}
		os.Exit(1)
	}
}
