package main

import (
	"os"

	"github.com/ishuar/tfskel/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
