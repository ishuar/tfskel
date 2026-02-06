package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/logger"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProjectStructure(t *testing.T) {
	t.Run("create structure with single region", func(t *testing.T) {
		baseDir := t.TempDir()
		log := logger.New(false)
		environments := []string{"dev", "stg", "prd"}
		err := createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, log)
		require.NoError(t, err)

		// Verify all root configuration files are created
		rootConfigFiles := []string{".gitignore", ".pre-commit-config.yaml", ".tflint.hcl", "trivy.yaml", ".tfskel.yaml"}
		for _, filename := range rootConfigFiles {
			filePath := filepath.Join(baseDir, filename)
			assert.FileExists(t, filePath, "Root config file %s should exist", filename)

			// Verify file is not empty
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "Root config file %s should not be empty", filename)
		}

		// Verify environment directories
		for _, env := range []string{"dev", "stg", "prd"} {
			envPath := filepath.Join(baseDir, "envs", env)

			// Verify .terraform-version file
			tfVersionPath := filepath.Join(envPath, ".terraform-version")
			assert.FileExists(t, tfVersionPath)
			content, err := os.ReadFile(tfVersionPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), "1.13.1", "Terraform version should be in file")

			// Verify region directory
			assert.DirExists(t, filepath.Join(envPath, "eu-central-1"))
		}
	})

	t.Run("create structure with multiple regions", func(t *testing.T) {
		baseDir := t.TempDir()
		log := logger.New(false)
		environments := []string{"dev", "stg", "prd"}
		regions := []string{"eu-central-1", "us-east-1", "ap-south-1"}
		err := createProjectStructure(baseDir, "1.10.0", regions, environments, log)
		require.NoError(t, err)

		// Verify all regions are created for all environments
		for _, env := range []string{"dev", "stg", "prd"} {
			for _, region := range regions {
				regionPath := filepath.Join(baseDir, "envs", env, region)
				assert.DirExists(t, regionPath, "Region directory %s/%s should exist", env, region)
			}
		}
	})

	t.Run("creates root config files via loop without errors", func(t *testing.T) {
		baseDir := t.TempDir()
		log := logger.New(false)
		environments := []string{"dev", "stg", "prd"}
		err := createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, log)
		require.NoError(t, err)

		// Test specifically for the refactored loop - ensure all files are created
		expectedFiles := map[string]string{
			".gitignore":              "root/.gitignore.tmpl",
			".pre-commit-config.yaml": "root/.pre-commit-config.yaml.tmpl",
			".tflint.hcl":             "root/.tflint.hcl.tmpl",
			"trivy.yaml":              "root/trivy.yaml.tmpl",
		}

		for filename := range expectedFiles {
			filePath := filepath.Join(baseDir, filename)
			assert.FileExists(t, filePath, "File %s from loop should exist", filename)

			// Verify files have actual content from templates
			stat, err := os.Stat(filePath)
			require.NoError(t, err)
			assert.Greater(t, stat.Size(), int64(0), "File %s should have content", filename)
		}
	})

	t.Run("handles existing root config files gracefully", func(t *testing.T) {
		baseDir := t.TempDir()
		log := logger.New(false)
		environments := []string{"dev", "stg", "prd"}

		// Create one file beforehand
		existingFile := filepath.Join(baseDir, ".gitignore")
		err := os.WriteFile(existingFile, []byte("# existing content"), 0644)
		require.NoError(t, err)

		// Run create structure
		err = createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, log)
		require.NoError(t, err)

		// Verify existing file wasn't overwritten
		content, err := os.ReadFile(existingFile)
		require.NoError(t, err)
		assert.Equal(t, "# existing content", string(content), "Existing file should not be overwritten")

		// Verify other files were still created
		assert.FileExists(t, filepath.Join(baseDir, ".pre-commit-config.yaml"))
		assert.FileExists(t, filepath.Join(baseDir, ".tflint.hcl"))
	})

	t.Run("creates correct terraform version in all environments", func(t *testing.T) {
		baseDir := t.TempDir()
		log := logger.New(false)
		version := "1.9.5"
		environments := []string{"dev", "stg", "prd"}

		err := createProjectStructure(baseDir, version, []string{"eu-central-1"}, environments, log)
		require.NoError(t, err)

		for _, env := range []string{"dev", "stg", "prd"} {
			tfVersionPath := filepath.Join(baseDir, "envs", env, ".terraform-version")
			content, err := os.ReadFile(tfVersionPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), version, "Version in %s should match", env)
		}
	})

	t.Run("create structure with custom environments", func(t *testing.T) {
		baseDir := t.TempDir()
		log := logger.New(false)
		customEnvs := []string{"dev", "qa", "uat", "prd"}
		err := createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, customEnvs, log)
		require.NoError(t, err)

		// Verify all custom environments are created
		for _, env := range customEnvs {
			envPath := filepath.Join(baseDir, "envs", env)
			tfVersionPath := filepath.Join(envPath, ".terraform-version")
			assert.FileExists(t, tfVersionPath, "Environment %s should have .terraform-version", env)

			// Verify region directory
			assert.DirExists(t, filepath.Join(envPath, "eu-central-1"))
		}

		// Verify no extra environments were created
		envsDir := filepath.Join(baseDir, "envs")
		entries, err := os.ReadDir(envsDir)
		require.NoError(t, err)
		assert.Equal(t, len(customEnvs), len(entries), "Should only have custom environments")
	})
}

func TestCreateFileFromTemplate(t *testing.T) {
	t.Run("create file from template with nil data", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "test", "file.txt")
		log := logger.New(false)

		err := createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil, log)
		require.NoError(t, err)

		assert.FileExists(t, targetPath)
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		assert.NotEmpty(t, string(content))
	})

	t.Run("create file from template with map data", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "test", "file.txt")
		log := logger.New(false)

		err := createFileFromTemplate(targetPath, "root/.terraform-version.tmpl", map[string]string{
			"TerraformVersion": "1.13.1",
		}, log)
		require.NoError(t, err)

		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "1.13.1")
	})

	t.Run("skips creation if file already exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "existing.txt")
		log := logger.New(false)

		// Create existing file
		existingContent := "original content"
		err := os.WriteFile(targetPath, []byte(existingContent), 0644)
		require.NoError(t, err)

		// Try to create from template
		err = createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil, log)
		require.NoError(t, err)

		// Verify file wasn't overwritten
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, existingContent, string(content))
	})

	t.Run("creates parent directories if they don't exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		targetPath := filepath.Join(tmpDir, "deep", "nested", "path", "file.txt")
		log := logger.New(false)

		err := createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil, log)
		require.NoError(t, err)

		assert.FileExists(t, targetPath)
		assert.DirExists(t, filepath.Join(tmpDir, "deep", "nested", "path"))
	})

	t.Run("handles multiple file creations in sequence", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		files := []string{".gitignore", ".tflint.hcl", "trivy.yaml"}
		for _, filename := range files {
			targetPath := filepath.Join(tmpDir, filename)
			err := createFileFromTemplate(targetPath, "root/"+filename+".tmpl", nil, log)
			require.NoError(t, err)
			assert.FileExists(t, targetPath)
		}
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	t.Run("creates config with defaults", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		log := logger.New(false)

		err := createDefaultConfig(configPath, log)
		require.NoError(t, err)

		assert.FileExists(t, configPath)

		content, err := os.ReadFile(configPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "terraform_version:")
		assert.Contains(t, contentStr, "provider:")
		assert.Contains(t, contentStr, "aws:")
		assert.Contains(t, contentStr, "version: ~> 6.0")
		assert.Contains(t, contentStr, "account_mapping:")
		assert.Contains(t, contentStr, "REPLACE_WITH_YOUR_DEV_ACCOUNT_ID")
		assert.Contains(t, contentStr, "backend:")
		assert.Contains(t, contentStr, "s3:")
		assert.Contains(t, contentStr, "bucket_name:")
		assert.Contains(t, contentStr, "default_tags:")
	})

	t.Run("skips if config exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		log := logger.New(false)

		// Create existing config
		err := os.WriteFile(configPath, []byte("existing"), 0644)
		require.NoError(t, err)

		err = createDefaultConfig(configPath, log)
		require.NoError(t, err)

		// Should not overwrite
		content, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, "existing", string(content))
	})
}

func TestDetermineInitParameters(t *testing.T) {
	t.Run("uses defaults when no config file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		envs, tfVersion, regions, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"dev", "stg", "prd"}, envs)
		assert.Equal(t, "1.13.1", tfVersion)
		assert.Equal(t, []string{"eu-central-1"}, regions)
	})

	t.Run("reads environments from existing config account_mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		// Create config with custom account mappings
		configContent := `terraform_version: "~> 1.13"
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
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		envs, tfVersion, regions, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		// Check that all environments from account_mapping are present
		assert.Len(t, envs, 4)
		assert.Contains(t, envs, "dev")
		assert.Contains(t, envs, "test")
		assert.Contains(t, envs, "qa")
		assert.Contains(t, envs, "prd")

		assert.Equal(t, "1.13.0", tfVersion)
		assert.Equal(t, []string{"us-east-1", "eu-west-1"}, regions)
	})

	t.Run("returns error when config exists but has no account_mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		// Create config without account_mapping
		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    regions:
      - "us-east-1"
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, err = determineInitParameters(tmpDir, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account_mapping is missing or empty")
	})

	t.Run("returns error when config exists with empty account_mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		// Create config with empty account_mapping
		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping: {}
    regions:
      - "us-east-1"
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, err = determineInitParameters(tmpDir, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account_mapping is missing or empty")
	})

	t.Run("uses defaults when config file is malformed", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		// Create malformed config
		configContent := `this is not: [valid yaml`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		envs, tfVersion, regions, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		// Should fall back to defaults
		assert.Equal(t, []string{"dev", "stg", "prd"}, envs)
		assert.Equal(t, "1.13.1", tfVersion)
		assert.Equal(t, []string{"eu-central-1"}, regions)
	})

	t.Run("extracts terraform version from constraint", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		configContent := `terraform_version: ">= 1.10.2"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, tfVersion, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, "1.10.2", tfVersion)
	})

	t.Run("uses default regions when not specified in config", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, regions, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"eu-central-1"}, regions)
	})
}

func TestRunInit(t *testing.T) {
	t.Run("init in current directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		t.Cleanup(func() {
			_ = os.Chdir(originalDir)
			initDir = ""
		})
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err = runInit(cmd, []string{})
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(tmpDir, ".gitignore"))
		assert.FileExists(t, filepath.Join(tmpDir, ".tfskel.yaml"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "stg"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "prd"))
	})

	t.Run("init with specific directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		initDir = tmpDir
		t.Cleanup(func() {
			initDir = ""
		})

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err := runInit(cmd, []string{})
		require.NoError(t, err)

		assert.FileExists(t, filepath.Join(tmpDir, ".gitignore"))
		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))
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
