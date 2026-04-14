package cmd

import (
	"testing"

	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadAndValidateConfig(t *testing.T) {
	t.Run("success with valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeTestConfig(t, tmpDir, "valid_config.yaml")
		chdirTemp(t, tmpDir)

		viper.Reset()
		initConfig()
		t.Cleanup(func() { viper.Reset() })

		cmd := newTestCmd(t)
		log := logger.New(false)

		cfg, err := loadAndValidateConfig(cmd, log)
		require.NoError(t, err)
		require.NotNil(t, cfg)
		assert.NotEmpty(t, cfg.TerraformVersion)
		assert.NotNil(t, cfg.Provider)
		assert.NotNil(t, cfg.Provider.AWS)
		assert.Equal(t, "test-tf-state-bucket", cfg.Backend.S3.BucketName)
		assert.Contains(t, cfg.Provider.AWS.AccountMapping, "dev")
	})

	t.Run("error when .tfskel.yaml is not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		chdirTemp(t, tmpDir)

		viper.Reset()
		initConfig()
		t.Cleanup(func() { viper.Reset() })

		cmd := newTestCmd(t)
		log := logger.New(false)

		_, err := loadAndValidateConfig(cmd, log)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrConfigNotFound)
	})

	t.Run("error when config missing required fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		writeTestConfigInline(t, tmpDir, `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    regions:
      - "us-east-1"
`)
		chdirTemp(t, tmpDir)

		viper.Reset()
		initConfig()
		t.Cleanup(func() { viper.Reset() })

		cmd := newTestCmd(t)
		log := logger.New(false)

		_, err := loadAndValidateConfig(cmd, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "configuration validation failed")
	})
}
