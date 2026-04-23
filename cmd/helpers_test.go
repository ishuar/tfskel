package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// writeTestConfig copies a testdata fixture file as .tfskel.yaml into dir.
// Use this for reusable config fixtures shared across multiple tests.
func writeTestConfig(t *testing.T, dir, fixtureName string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", fixtureName))
	require.NoError(t, err, "failed to read testdata/%s", fixtureName)
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".tfskel.yaml"), data, 0644))
}

// writeTestConfigInline writes inline YAML content as .tfskel.yaml into dir.
// Use for one-off edge cases (malformed YAML, minimal configs) that don't warrant a fixture file.
func writeTestConfigInline(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".tfskel.yaml"), []byte(content), 0644))
}

// chdirTemp changes into dir and restores the original working directory on cleanup.
func chdirTemp(t *testing.T, dir string) {
	t.Helper()
	origDir, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })
}

// newTestCmd creates a minimal cobra.Command with the "config" flag that
// loadAndValidateConfig looks for, plus scaffold-specific flags commonly
// needed by downstream config loading.
func newTestCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file")
	cmd.Flags().String("templates-dir", "", "")
	cmd.Flags().String("s3-bucket-name", "", "")
	return cmd
}
