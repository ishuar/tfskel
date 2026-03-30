package toolcheck

import (
	"context"
	"os/exec"
	"time"
)

// CommandRunner abstracts exec.LookPath and exec.Command for testability.
type CommandRunner interface {
	LookPath(file string) (string, error)
	RunCommand(name string, args ...string) (string, error)
}

// commandTimeout is the maximum time to wait for a tool version check command.
const commandTimeout = 5 * time.Second

// OSCommandRunner implements CommandRunner using real os/exec calls.
type OSCommandRunner struct{}

// LookPath searches for an executable named file in the directories named by the PATH.
func (r *OSCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// RunCommand executes a command with a 5-second timeout and returns its combined output.
func (r *OSCommandRunner) RunCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}
