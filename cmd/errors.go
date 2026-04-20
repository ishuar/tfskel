package cmd

import (
	"errors"
	"fmt"
)

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

// Flag-validation sentinel errors shared across commands.
var (
	// ErrForceRequiresUpgrade indicates --force was used without --upgrade.
	ErrForceRequiresUpgrade = errors.New("--force can only be used together with --upgrade")
	// ErrInitSkipRequiresUpgrade indicates --skip was used without --upgrade in init.
	ErrInitSkipRequiresUpgrade = errors.New("--skip can only be used with --upgrade")
	// ErrScaffoldForceRequiresUpgrade indicates --force was used without --upgrade or --upgrade-all in scaffold.
	ErrScaffoldForceRequiresUpgrade = errors.New("--force can only be used together with --upgrade or --upgrade-all")
	// ErrUpgradeAllWithAppDir indicates --upgrade-all was used with a positional <app-dir> argument.
	ErrUpgradeAllWithAppDir = errors.New("cannot specify <app-dir> when --upgrade-all is set")
	// ErrUpgradeAllWithUpgrade indicates --upgrade-all and --upgrade were both set.
	ErrUpgradeAllWithUpgrade = errors.New("--upgrade-all and --upgrade are mutually exclusive")
	// ErrSkipRequiresUpgradeAll indicates --skip was used without --upgrade-all.
	ErrSkipRequiresUpgradeAll = errors.New("--skip can only be used with --upgrade-all")
	// ErrBaseDirNotExist indicates the envs/<env>/<region>/ directory does not exist.
	ErrBaseDirNotExist = errors.New("base directory does not exist; check --env and --region values")
	// ErrNoAppDirsFound indicates no app directories were found for --upgrade-all.
	ErrNoAppDirsFound = errors.New("no app directories found")
)
