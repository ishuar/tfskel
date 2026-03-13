package cmd

import "fmt"

// ExitError represents an error that should cause the program to exit with a specific code
type ExitError struct {
	Code    int
	Message string
}

func (e *ExitError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("exit code %d", e.Code)
}

// NewExitError creates a new ExitError with the given code and message
func NewExitError(code int, message string) *ExitError {
	return &ExitError{Code: code, Message: message}
}
