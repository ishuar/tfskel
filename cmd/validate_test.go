package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validateTestSetup creates a project directory with a valid config and
// initialized structure so runValidate can run its checks. Returns the project
// directory path. Viper is reset per-call and cleaned up on test completion.
func validateTestSetup(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	writeTestConfig(t, tmpDir, "valid_config.yaml")
	chdirTemp(t, tmpDir)

	// Run init to create the project structure that validate expects.
	initOpts := &initOpts{root: &rootOpts{}, dir: tmpDir}
	require.NoError(t, initOpts.run(newTestCmd(t), []string{}))

	// Set up viper for the validate command (re-read config).
	viper.Reset()
	(&rootOpts{}).initConfig()
	t.Cleanup(func() { viper.Reset() })

	return tmpDir
}

func TestRunValidate(t *testing.T) {
	t.Run("runs all checks on valid project", func(t *testing.T) {
		validateTestSetup(t)
		opts := &validateOpts{root: &rootOpts{}, format: "table"}

		err := opts.run(newTestCmd(t), []string{})
		// Validate may return ExitError if tools check fails (tools not installed in CI),
		// but config check should pass. Accept both nil and ExitError.
		if err != nil {
			var exitErr *ExitError
			require.True(t, errors.As(err, &exitErr), "error should be ExitError, got: %v", err)
		}
	})

	t.Run("skip tools check", func(t *testing.T) {
		validateTestSetup(t)
		opts := &validateOpts{root: &rootOpts{}, format: "table", skip: "tools"}

		err := opts.run(newTestCmd(t), []string{})
		assert.NoError(t, err)
	})

	t.Run("skip config check", func(t *testing.T) {
		validateTestSetup(t)
		opts := &validateOpts{root: &rootOpts{}, format: "table", skip: "config"}

		err := opts.run(newTestCmd(t), []string{})
		if err != nil {
			var exitErr *ExitError
			require.True(t, errors.As(err, &exitErr), "error should be ExitError, got: %v", err)
		}
	})

	t.Run("JSON output format", func(t *testing.T) {
		validateTestSetup(t)
		opts := &validateOpts{root: &rootOpts{}, format: "json", skip: "tools"}

		err := opts.run(newTestCmd(t), []string{})
		assert.NoError(t, err)
	})

	t.Run("CSV output format", func(t *testing.T) {
		validateTestSetup(t)
		opts := &validateOpts{root: &rootOpts{}, format: "csv", skip: "tools"}

		err := opts.run(newTestCmd(t), []string{})
		assert.NoError(t, err)
	})

	t.Run("invalid skip value returns error", func(t *testing.T) {
		validateTestSetup(t)
		opts := &validateOpts{root: &rootOpts{}, skip: "bogus"}

		err := opts.run(newTestCmd(t), []string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bogus")
	})

	t.Run("fails without valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		chdirTemp(t, tmpDir)

		viper.Reset()
		(&rootOpts{}).initConfig()
		t.Cleanup(func() { viper.Reset() })

		opts := &validateOpts{root: &rootOpts{}, format: "table"}
		err := opts.run(newTestCmd(t), []string{})
		require.Error(t, err)
		require.ErrorIs(t, err, ErrConfigNotFound)
	})
}

func TestValidateCmd_CommandSetup(t *testing.T) {
	root := NewRootCmd()
	validateCmd, _, err := root.Find([]string{"validate"})
	require.NoError(t, err)

	t.Run("command is properly registered", func(t *testing.T) {
		assert.Equal(t, "validate", validateCmd.Use)
		assert.Equal(t, "main", validateCmd.GroupID)
		assert.NotEmpty(t, validateCmd.Short)
	})

	t.Run("has required flags", func(t *testing.T) {
		formatFlag := validateCmd.Flags().Lookup("format")
		require.NotNil(t, formatFlag)
		assert.Equal(t, "f", formatFlag.Shorthand)
		assert.Equal(t, "table", formatFlag.DefValue)

		skipFlag := validateCmd.Flags().Lookup("skip")
		require.NotNil(t, skipFlag)
		assert.Equal(t, "", skipFlag.DefValue)
	})
}

// captureStdout swaps os.Stdout for a pipe for the duration of fn and returns
// what was written. Used to inspect JSON output from runValidate, which writes
// directly to os.Stdout rather than an injected writer.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	done := make(chan []byte)
	go func() {
		b, _ := io.ReadAll(r)
		done <- b
	}()

	fn()

	require.NoError(t, w.Close())
	os.Stdout = orig
	return string(<-done)
}

// TestRunValidate_RelativeConfigPath verifies that when --config is passed as a
// relative path, the report's ProjectRoot/ConfigPath are still absolute. This
// guards the filepath.Abs normalization in runValidate.
func TestRunValidate_RelativeConfigPath(t *testing.T) {
	tmpDir := validateTestSetup(t)

	// Re-initialize viper using a relative --config path. ConfigFileUsed() will
	// then be relative, exercising the abs-resolution branch in runValidate.
	viper.Reset()
	(&rootOpts{cfgFile: "./.tfskel.yaml"}).initConfig()
	t.Cleanup(func() { viper.Reset() })
	require.False(t, filepath.IsAbs(viper.ConfigFileUsed()),
		"precondition: viper should retain the relative --config path")

	opts := &validateOpts{root: &rootOpts{}, format: "json", skip: "tools"}
	out := captureStdout(t, func() {
		err := opts.run(newTestCmd(t), []string{})
		require.NoError(t, err)
	})

	var report struct {
		ProjectRoot string `json:"projectRoot"`
		ConfigPath  string `json:"configPath"`
	}
	require.NoError(t, json.Unmarshal([]byte(out), &report))
	assert.True(t, filepath.IsAbs(report.ProjectRoot),
		"projectRoot should be absolute, got %q", report.ProjectRoot)
	assert.True(t, filepath.IsAbs(report.ConfigPath),
		"configPath should be absolute, got %q", report.ConfigPath)

	// On macOS, t.TempDir() returns /var/... but os.Getwd() canonicalizes
	// through the /private symlink. Compare against the resolved path.
	canonical, err := filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, canonical, report.ProjectRoot)
	assert.Equal(t, filepath.Join(canonical, ".tfskel.yaml"), report.ConfigPath)
}

func TestRunValidate_SkipToolsConfigPass(t *testing.T) {
	// Integration test: a freshly init'd project with matching .terraform-version
	// files should produce a passing config check.
	tmpDir := validateTestSetup(t)

	assert.FileExists(t, filepath.Join(tmpDir, "envs", "dev", ".terraform-version"))
	assert.FileExists(t, filepath.Join(tmpDir, "envs", "stg", ".terraform-version"))
	assert.FileExists(t, filepath.Join(tmpDir, "envs", "prd", ".terraform-version"))

	opts := &validateOpts{root: &rootOpts{}, format: "table", skip: "tools"}
	err := opts.run(newTestCmd(t), []string{})
	assert.NoError(t, err, "config check should pass on freshly init'd project")
}
