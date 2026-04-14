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

// saveAndRestoreScaffoldFlags saves all scaffold package-level flag vars
// and restores them on test cleanup.
func saveAndRestoreScaffoldFlags(t *testing.T) {
	t.Helper()
	origEnv := env
	origRegion := region
	origUpgrade := scaffoldUpgrade
	origForce := scaffoldForce
	origUpgradeAll := scaffoldUpgradeAll
	origSkip := scaffoldSkip
	origDryRun := dryRun
	origUseColor := useColor
	origWorkflowsEnv := workflowsEnv
	origWorkflowUpgrade := workflowUpgrade
	origWorkflowForce := workflowForce
	t.Cleanup(func() {
		env = origEnv
		region = origRegion
		scaffoldUpgrade = origUpgrade
		scaffoldForce = origForce
		scaffoldUpgradeAll = origUpgradeAll
		scaffoldSkip = origSkip
		dryRun = origDryRun
		useColor = origUseColor
		workflowsEnv = origWorkflowsEnv
		workflowUpgrade = origWorkflowUpgrade
		workflowForce = origWorkflowForce
	})
}

// saveAndRestoreInitFlags saves all init package-level flag vars
// and restores them on test cleanup.
func saveAndRestoreInitFlags(t *testing.T) {
	t.Helper()
	origDir := initDir
	origWorkflows := initWorkflows
	origUpgrade := initUpgrade
	origForce := initForce
	origDryRun := dryRun
	origUseColor := useColor
	t.Cleanup(func() {
		initDir = origDir
		initWorkflows = origWorkflows
		initUpgrade = origUpgrade
		initForce = origForce
		dryRun = origDryRun
		useColor = origUseColor
	})
}

// newTestCmd creates a minimal cobra.Command with a "config" flag,
// which is needed by loadAndValidateConfig.
func newTestCmd(t *testing.T) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file")
	cmd.Flags().String("templates-dir", "", "")
	cmd.Flags().String("s3-bucket-name", "", "")
	return cmd
}
