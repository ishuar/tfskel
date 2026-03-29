package toolcheck

import (
	"os/exec"
)

// CommandRunner abstracts exec.LookPath and exec.Command for testability.
type CommandRunner interface {
	LookPath(file string) (string, error)
	RunCommand(name string, args ...string) (string, error)
}

// OSCommandRunner implements CommandRunner using real os/exec calls.
type OSCommandRunner struct{}

// LookPath searches for an executable named file in the directories named by the PATH.
func (r *OSCommandRunner) LookPath(file string) (string, error) {
	return exec.LookPath(file)
}

// RunCommand executes a command and returns its combined stdout and stderr output.
func (r *OSCommandRunner) RunCommand(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}
