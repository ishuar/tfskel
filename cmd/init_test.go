package cmd

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitSkipRequiresUpgrade(t *testing.T) {
	tmpDir := t.TempDir()
	saveAndRestoreInitFlags(t)
	initDir = tmpDir
	initSkip = "trivy.yaml"

	err := runInit(newTestCmd(t), []string{})
	assert.ErrorIs(t, err, ErrInitSkipRequiresUpgrade)
}

func TestRunInit(t *testing.T) {
	t.Run("init in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		chdirTemp(t, tmpDir)
		saveAndRestoreInitFlags(t)

		err := runInit(newTestCmd(t), []string{})
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(tmpDir, ".gitignore"))
		assert.FileExists(t, filepath.Join(tmpDir, ".tfskel.yaml"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "stg"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "prd"))
	})

	t.Run("init with specific directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		saveAndRestoreInitFlags(t)
		initDir = tmpDir

		err := runInit(newTestCmd(t), []string{})
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(tmpDir, ".gitignore"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))
	})

	t.Run("init respects existing config workflows.create false", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeTestConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
      stg: "234567890123"
      prd: "345678901234"
    regions:
      - "eu-central-1"
backend:
  s3:
    bucket_name: "test-tf-state-bucket"
workflows:
  create: false
  name: "terraform"
  aws_role_name: "terraform-role"
`)
		saveAndRestoreInitFlags(t)
		initDir = tmpDir

		err := runInit(newTestCmd(t), []string{})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))

		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		assert.NoDirExists(t, workflowsDir, "workflows should not be created when config has create: false")
	})

	t.Run("init CLI flag overrides config workflows.create", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeTestConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
      stg: "234567890123"
      prd: "345678901234"
    regions:
      - "eu-central-1"
backend:
  s3:
    bucket_name: "test-tf-state-bucket"
workflows:
  create: false
  name: "terraform"
  aws_role_name: "terraform-role"
`)
		saveAndRestoreInitFlags(t)
		initDir = tmpDir
		initWorkflows = true

		err := runInit(newTestCmd(t), []string{})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))

		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		assert.DirExists(t, workflowsDir, "workflows should be created when CLI flag overrides config")
		assert.FileExists(t, filepath.Join(workflowsDir, "lint.yaml"))
	})
}

func TestInitCmd(t *testing.T) {
	assert.NotNil(t, initCmd)
	assert.Equal(t, "init", initCmd.Use)
	assert.NotEmpty(t, initCmd.Short)

	dirFlag := initCmd.Flags().Lookup("dir")
	assert.NotNil(t, dirFlag)
	assert.Equal(t, "d", dirFlag.Shorthand)
}

func TestRunInit_FlagValidation(t *testing.T) {
	t.Run("force without upgrade returns error", func(t *testing.T) {
		tmpDir := t.TempDir()
		saveAndRestoreInitFlags(t)
		initDir = tmpDir
		initForce = true
		initUpgrade = false

		err := runInit(newTestCmd(t), []string{})
		assert.ErrorIs(t, err, ErrForceRequiresUpgrade)
	})
}

func TestRunInit_DryRun(t *testing.T) {
	t.Run("dry-run creates no files", func(t *testing.T) {
		tmpDir := t.TempDir()
		saveAndRestoreInitFlags(t)
		initDir = tmpDir
		dryRun = true

		err := runInit(newTestCmd(t), []string{})
		require.NoError(t, err)

		assert.NoFileExists(t, filepath.Join(tmpDir, ".gitignore"), "dry-run should not create files")
		assert.NoDirExists(t, filepath.Join(tmpDir, "envs"), "dry-run should not create directories")
	})
}
