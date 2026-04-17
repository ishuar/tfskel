package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfigInline writes inline YAML content as .tfskel.yaml into dir.
func writeConfigInline(t *testing.T, dir, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".tfskel.yaml"), []byte(content), 0644))
}

func TestDetermineParameters(t *testing.T) {
	t.Run("uses defaults when no config file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"dev", "stg", "prd"}, params.Environments)
		assert.Equal(t, "1.13.1", params.TerraformVersion)
		assert.Equal(t, []string{"eu-central-1"}, params.Regions)
	})

	t.Run("reads environments from existing config account_mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		writeConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "111111111111"
      test: "222222222222"
      qa: "333333333333"
      prd: "444444444444"
    regions:
      - "us-east-1"
      - "eu-west-1"
`)

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Len(t, params.Environments, 4)
		assert.Contains(t, params.Environments, "dev")
		assert.Contains(t, params.Environments, "test")
		assert.Contains(t, params.Environments, "qa")
		assert.Contains(t, params.Environments, "prd")

		assert.Equal(t, "1.13.0", params.TerraformVersion)
		assert.Equal(t, []string{"us-east-1", "eu-west-1"}, params.Regions)
	})

	t.Run("returns error when config exists but has no account_mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		writeConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    regions:
      - "us-east-1"
`)

		_, err := DetermineParameters(tmpDir, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account_mapping is missing or empty")
	})

	t.Run("uses defaults when config file is malformed", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		writeConfigInline(t, tmpDir, `this is not: [valid yaml`)

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"dev", "stg", "prd"}, params.Environments)
		assert.Equal(t, defaultTerraformVersion, params.TerraformVersion)
		assert.Equal(t, []string{"eu-central-1"}, params.Regions)
	})

	t.Run("extracts terraform version from constraint", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		writeConfigInline(t, tmpDir, `terraform_version: ">= 1.10.2"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`)

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, "1.10.2", params.TerraformVersion)
	})

	t.Run("uses default regions when not specified in config", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		writeConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`)

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"eu-central-1"}, params.Regions)
	})
}

func TestDetermineParametersWorkflows(t *testing.T) {
	log := logger.New(false)

	t.Run("returns false when config has workflows.create: false", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeConfigInline(t, tmpDir, `workflows:
  create: false
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`)
		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, params.CreateWorkflows)
	})

	t.Run("returns true when config has workflows.create: true", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeConfigInline(t, tmpDir, `workflows:
  create: true
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`)
		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)
		assert.True(t, params.CreateWorkflows)
	})

	t.Run("returns false when no config file exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, params.CreateWorkflows)
	})

	t.Run("returns false when config has no workflows section", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`)
		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, params.CreateWorkflows)
	})

	t.Run("returns false when config file is malformed", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeConfigInline(t, tmpDir, `this is not: [valid yaml`)

		params, err := DetermineParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, params.CreateWorkflows)
	})
}
