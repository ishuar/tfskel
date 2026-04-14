package cmd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateTestSetup creates a project directory with a valid config and
// initialized structure (envs/, .terraform-version files, etc.) so that
// runValidate can run its checks. Returns the project directory path.
func validateTestSetup(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, "valid_config.yaml")
	chdirTemp(t, tmpDir)
	saveAndRestoreInitFlags(t)
	saveAndRestoreValidateFlags(t)

	// Run init to create the project structure that validate expects.
	initDir = tmpDir
	err := runInit(newTestCmd(t), []string{})
	require.NoError(t, err)

	// Now set up viper for the validate command (re-read config).
	viper.Reset()
	initConfig()
	t.Cleanup(func() { viper.Reset() })

	useColor = false
	return tmpDir
}

// saveAndRestoreValidateFlags saves/restores validate package-level flags.
func saveAndRestoreValidateFlags(t *testing.T) {
	t.Helper()
	origFormat := validateFormat
	origSkip := validateSkip
	origDryRun := dryRun
	origUseColor := useColor
	t.Cleanup(func() {
		validateFormat = origFormat
		validateSkip = origSkip
		dryRun = origDryRun
		useColor = origUseColor
	})
}

func TestRunValidate(t *testing.T) {
	t.Run("runs all checks on valid project", func(t *testing.T) {
		validateTestSetup(t)
		validateFormat = "table"
		validateSkip = ""

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		// Validate may return ExitError if tools check fails (tools not installed in CI),
		// but config check should pass. Accept both nil and ExitError.
		if err != nil {
			var exitErr *ExitError
			require.True(t, errors.As(err, &exitErr), "error should be ExitError, got: %v", err)
		}
	})

	t.Run("skip tools check", func(t *testing.T) {
		validateTestSetup(t)
		validateFormat = "table"
		validateSkip = "tools"

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		// With tools skipped and a freshly init'd project, config check should pass
		assert.NoError(t, err)
	})

	t.Run("skip config check", func(t *testing.T) {
		validateTestSetup(t)
		validateFormat = "table"
		validateSkip = "config"

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		// Tools check may fail if tools aren't installed, that's OK
		if err != nil {
			var exitErr *ExitError
			require.True(t, errors.As(err, &exitErr), "error should be ExitError, got: %v", err)
		}
	})

	t.Run("JSON output format", func(t *testing.T) {
		validateTestSetup(t)
		validateFormat = "json"
		validateSkip = "tools"

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("CSV output format", func(t *testing.T) {
		validateTestSetup(t)
		validateFormat = "csv"
		validateSkip = "tools"

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("invalid skip value returns error", func(t *testing.T) {
		validateTestSetup(t)
		validateSkip = "bogus"

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bogus")
	})

	t.Run("fails without valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		chdirTemp(t, tmpDir)
		saveAndRestoreValidateFlags(t)

		viper.Reset()
		initConfig()
		t.Cleanup(func() { viper.Reset() })

		validateFormat = "table"
		validateSkip = ""

		cmd := newTestCmd(t)
		err := runValidate(cmd, []string{})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrConfigNotFound)
	})
}

func TestValidateCmd_CommandSetup(t *testing.T) {
	t.Run("command is properly registered", func(t *testing.T) {
		assert.NotNil(t, validateCmd)
		assert.Equal(t, "validate", validateCmd.Use)
		assert.Equal(t, "main", validateCmd.GroupID)
		assert.NotEmpty(t, validateCmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		formatFlag := validateCmd.Flags().Lookup("format")
		assert.NotNil(t, formatFlag)
		assert.Equal(t, "f", formatFlag.Shorthand)
		assert.Equal(t, "table", formatFlag.DefValue)

		skipFlag := validateCmd.Flags().Lookup("skip")
		assert.NotNil(t, skipFlag)
		assert.Equal(t, "", skipFlag.DefValue)
	})
}

func TestRunValidate_SkipToolsConfigPass(t *testing.T) {
	// Integration test: a freshly init'd project with matching .terraform-version
	// files should produce a passing config check.
	tmpDir := validateTestSetup(t)
	validateFormat = "table"
	validateSkip = "tools"

	// Verify the structure that validate will scan
	assert.FileExists(t, filepath.Join(tmpDir, "envs", "dev", ".terraform-version"))
	assert.FileExists(t, filepath.Join(tmpDir, "envs", "stg", ".terraform-version"))
	assert.FileExists(t, filepath.Join(tmpDir, "envs", "prd", ".terraform-version"))

	cmd := newTestCmd(t)
	err := runValidate(cmd, []string{})
	assert.NoError(t, err, "config check should pass on freshly init'd project")
}
