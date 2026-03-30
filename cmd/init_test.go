package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestInitRunner creates an initRunner with MemoryFileSystem for unit tests.
func newTestInitRunner(t *testing.T) *initRunner {
	t.Helper()
	renderer, err := templates.NewRenderer()
	require.NoError(t, err)
	return &initRunner{
		fs:       fs.NewMemoryFileSystem(),
		log:      logger.New(false),
		renderer: renderer,
	}
}

func TestCreateProjectStructure(t *testing.T) {
	t.Run("create structure with single region", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		// Verify all root configuration files are created
		for _, filename := range []string{".gitignore", ".pre-commit-config.yaml", ".tflint.hcl", "trivy.yaml", ".tfskel.yaml", ".mise.toml"} {
			filePath := filepath.Join(baseDir, filename)
			assert.True(t, r.fs.FileExists(filePath), "Root config file %s should exist", filename)

			content, err := r.fs.ReadFile(filePath)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "Root config file %s should not be empty", filename)
		}

		// Verify environment directories
		for _, env := range environments {
			envPath := filepath.Join(baseDir, "envs", env)

			tfVersionPath := filepath.Join(envPath, ".terraform-version")
			assert.True(t, r.fs.FileExists(tfVersionPath))
			content, err := r.fs.ReadFile(tfVersionPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), "1.13.1")

			assert.True(t, r.fs.DirExists(filepath.Join(envPath, "eu-central-1")))
		}
	})

	t.Run("create structure with multiple regions", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}
		regions := []string{"eu-central-1", "us-east-1", "ap-south-1"}

		err := r.createProjectStructure(baseDir, "1.10.0", regions, environments, true)
		require.NoError(t, err)

		for _, env := range environments {
			for _, region := range regions {
				regionPath := filepath.Join(baseDir, "envs", env, region)
				assert.True(t, r.fs.DirExists(regionPath), "Region directory %s/%s should exist", env, region)
			}
		}
	})

	t.Run("creates root config files via loop without errors", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		for _, filename := range []string{".gitignore", ".pre-commit-config.yaml", ".tflint.hcl", "trivy.yaml"} {
			filePath := filepath.Join(baseDir, filename)
			assert.True(t, r.fs.FileExists(filePath), "File %s from loop should exist", filename)

			content, err := r.fs.ReadFile(filePath)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "File %s should have content", filename)
		}
	})

	t.Run("handles existing root config files gracefully", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}

		// Create one file beforehand
		existingFile := filepath.Join(baseDir, ".gitignore")
		require.NoError(t, r.fs.WriteFile(existingFile, []byte("# existing content"), 0644))

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		// Verify existing file wasn't overwritten
		content, err := r.fs.ReadFile(existingFile)
		require.NoError(t, err)
		assert.Equal(t, "# existing content", string(content))

		// Verify other files were still created
		assert.True(t, r.fs.FileExists(filepath.Join(baseDir, ".pre-commit-config.yaml")))
		assert.True(t, r.fs.FileExists(filepath.Join(baseDir, ".tflint.hcl")))
	})

	t.Run("creates correct terraform version in all environments", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"
		version := "1.9.5"
		environments := []string{"dev", "stg", "prd"}

		err := r.createProjectStructure(baseDir, version, []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		for _, env := range environments {
			tfVersionPath := filepath.Join(baseDir, "envs", env, ".terraform-version")
			content, err := r.fs.ReadFile(tfVersionPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), version, "Version in %s should match", env)
		}
	})

	t.Run("create structure with custom environments", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"
		customEnvs := []string{"dev", "qa", "uat", "prd"}

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, customEnvs, true)
		require.NoError(t, err)

		for _, env := range customEnvs {
			envPath := filepath.Join(baseDir, "envs", env)
			assert.True(t, r.fs.FileExists(filepath.Join(envPath, ".terraform-version")), "Environment %s should have .terraform-version", env)
			assert.True(t, r.fs.DirExists(filepath.Join(envPath, "eu-central-1")))
		}
	})

	t.Run("creates static workflow files when createWorkflows is true", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, true)
		require.NoError(t, err)

		for _, f := range []string{"lint.yaml", "reusable-detect-changes.yaml", "reusable-terraform-plan-apply.yaml", "reusable-lint.yaml"} {
			p := filepath.Join(baseDir, ".github", "workflows", f)
			assert.True(t, r.fs.FileExists(p), "static workflow file %s should exist", f)
			content, readErr := r.fs.ReadFile(p)
			require.NoError(t, readErr)
			assert.NotEmpty(t, content, "static workflow file %s should not be empty", f)
		}
	})

	t.Run("skips workflow files when createWorkflows is false", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		workflowsDir := filepath.Join(baseDir, ".github", "workflows")
		assert.False(t, r.fs.DirExists(workflowsDir), ".github/workflows directory should not be created when disabled")
	})

	t.Run("dry-run does not create any files or directories", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		dryFS := fs.NewDryRunFileSystem(memFS)
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)

		r := &initRunner{
			fs:       dryFS,
			log:      logger.New(false),
			renderer: renderer,
			dryRun:   true,
		}
		baseDir := "/project"

		err = r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev", "stg", "prd"}, true)
		require.NoError(t, err)

		// The underlying MemoryFileSystem should have no files or directories
		assert.False(t, memFS.FileExists(filepath.Join(baseDir, ".gitignore")))
		assert.False(t, memFS.FileExists(filepath.Join(baseDir, ".tfskel.yaml")))
		assert.False(t, memFS.DirExists(filepath.Join(baseDir, "envs", "dev", "eu-central-1")))
		assert.False(t, memFS.FileExists(filepath.Join(baseDir, ".github", "workflows", "lint.yaml")))
	})

	t.Run("does not overwrite existing static workflow files", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		wfDir := filepath.Join(baseDir, ".github", "workflows")
		require.NoError(t, r.fs.MkdirAll(wfDir, 0755))
		existingPath := filepath.Join(wfDir, "lint.yaml")
		require.NoError(t, r.fs.WriteFile(existingPath, []byte("# custom lint"), 0644))

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, true)
		require.NoError(t, err)

		content, readErr := r.fs.ReadFile(existingPath)
		require.NoError(t, readErr)
		assert.Equal(t, "# custom lint", string(content), "existing workflow file should not be overwritten")
	})

	t.Run("creates .mise.toml with correct terraform version", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		misePath := filepath.Join(baseDir, ".mise.toml")
		assert.True(t, r.fs.FileExists(misePath), ".mise.toml should be created")

		content, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, `terraform = "1.13.1"`)
		assert.Contains(t, contentStr, "min_version")
	})

	t.Run(".mise.toml uses latest for tools when no tools config", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, ".mise.toml"))
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, `tflint = "latest"`)
		assert.Contains(t, contentStr, `trivy = "latest"`)
		assert.Contains(t, contentStr, `pre-commit = "latest"`)
		assert.Contains(t, contentStr, `awscli = "latest"`)
	})

	t.Run(".mise.toml respects pinned tool versions from config", func(t *testing.T) {
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		r := &initRunner{
			fs:       fs.NewMemoryFileSystem(),
			log:      logger.New(false),
			renderer: renderer,
			tools: map[string]string{
				"tflint": "0.50.0",
				"trivy":  "0.58.2",
			},
		}
		baseDir := "/project"

		err = r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, ".mise.toml"))
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, `tflint = "0.50.0"`, "pinned tflint version should be used")
		assert.Contains(t, contentStr, `trivy = "0.58.2"`, "pinned trivy version should be used")
		assert.Contains(t, contentStr, `pre-commit = "latest"`, "unpinned tools should default to latest")
		assert.Contains(t, contentStr, `terraform = "1.13.1"`, "terraform version from config should be used")
	})

	t.Run(".mise.toml strips terraform version constraint", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		// Pass a constraint-style version (already stripped by determineInitParameters in real usage,
		// but test that stripConstraint in the template handles it)
		err := r.createProjectStructure(baseDir, "~> 1.13", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, ".mise.toml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), `terraform = "1.13"`, "constraint should be stripped")
		assert.NotContains(t, string(content), "~>", "constraint operator should not appear in .mise.toml")
	})

	t.Run("does not overwrite existing .mise.toml", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		misePath := filepath.Join(baseDir, ".mise.toml")
		require.NoError(t, r.fs.WriteFile(misePath, []byte("# custom mise config"), 0644))

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		assert.Equal(t, "# custom mise config", string(content), "existing .mise.toml should not be overwritten")
	})
}

func TestCreateFileFromTemplate(t *testing.T) {
	t.Run("create file from template with nil data", func(t *testing.T) {
		r := newTestInitRunner(t)
		targetPath := "/project/test/file.txt"

		err := r.createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(targetPath))
		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.NotEmpty(t, string(content))
	})

	t.Run("create file from template with data", func(t *testing.T) {
		r := newTestInitRunner(t)
		targetPath := "/project/test/file.txt"

		err := r.createFileFromTemplate(targetPath, "root/.terraform-version.tmpl", &templates.Data{
			TerraformVersion: "1.13.1",
		})
		require.NoError(t, err)

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "1.13.1")
	})

	t.Run("skips creation if file already exists", func(t *testing.T) {
		r := newTestInitRunner(t)
		targetPath := "/project/existing.txt"

		existingContent := "original content"
		require.NoError(t, r.fs.WriteFile(targetPath, []byte(existingContent), 0644))

		err := r.createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, existingContent, string(content))
	})

	t.Run("creates file in deeply nested path", func(t *testing.T) {
		r := newTestInitRunner(t)
		targetPath := "/project/deep/nested/path/file.txt"

		err := r.createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(targetPath))
	})

	t.Run("handles multiple file creations in sequence", func(t *testing.T) {
		r := newTestInitRunner(t)

		files := []string{".gitignore", ".tflint.hcl", "trivy.yaml"}
		for _, filename := range files {
			targetPath := filepath.Join("/project", filename)
			err := r.createFileFromTemplate(targetPath, "root/"+filename+".tmpl", nil)
			require.NoError(t, err)
			assert.True(t, r.fs.FileExists(targetPath))
		}
	})
}

func TestCreateDefaultConfig(t *testing.T) {
	t.Run("creates config with defaults", func(t *testing.T) {
		r := newTestInitRunner(t)
		configPath := "/project/.tfskel.yaml"

		err := r.createDefaultConfig(configPath)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(configPath))

		content, err := r.fs.ReadFile(configPath)
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
		assert.Contains(t, contentStr, "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME")
		assert.Contains(t, contentStr, "default_tags:")
		assert.Contains(t, contentStr, "critical_resources:")
		assert.Contains(t, contentStr, "For full configuration reference")
	})

	t.Run("skips if config exists", func(t *testing.T) {
		r := newTestInitRunner(t)
		configPath := "/project/.tfskel.yaml"

		require.NoError(t, r.fs.WriteFile(configPath, []byte("existing"), 0644))

		err := r.createDefaultConfig(configPath)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, "existing", string(content))
	})
}

func TestDetermineInitParameters(t *testing.T) {
	t.Run("uses defaults when no config file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		envs, tfVersion, regions, _, _, err := determineInitParameters(tmpDir, log)
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

		envs, tfVersion, regions, _, _, err := determineInitParameters(tmpDir, log)
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

		_, _, _, _, _, err = determineInitParameters(tmpDir, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account_mapping is missing or empty")
	})

	t.Run("returns error when config exists with empty account_mapping", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

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

		_, _, _, _, _, err = determineInitParameters(tmpDir, log)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account_mapping is missing or empty")
	})

	t.Run("uses defaults when config file is malformed", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		configContent := `this is not: [valid yaml`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		envs, tfVersion, regions, _, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"dev", "stg", "prd"}, envs)
		assert.Equal(t, defaultTerraformVersion, tfVersion)
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

		_, tfVersion, _, _, _, err := determineInitParameters(tmpDir, log)
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

		_, _, regions, _, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, []string{"eu-central-1"}, regions)
	})

	t.Run("returns tools map from config", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
tools:
  tflint: "0.50.0"
  trivy: "0.58.2"
`
		err := os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, _, tools, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Equal(t, "0.50.0", tools["tflint"])
		assert.Equal(t, "0.58.2", tools["trivy"])
	})

	t.Run("returns nil tools when config has no tools section", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		err := os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, _, tools, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Nil(t, tools, "tools should be nil when not configured")
	})

	t.Run("returns nil tools when no config file exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		log := logger.New(false)

		_, _, _, _, tools, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)

		assert.Nil(t, tools)
	})
}

func TestDetermineInitParametersWorkflows(t *testing.T) {
	log := logger.New(false)

	t.Run("returns false when config has workflows.create: false", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `workflows:
  create: false
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		err := os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, createWorkflows, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, createWorkflows)
	})

	t.Run("returns true when config has workflows.create: true", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `workflows:
  create: true
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		err := os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, createWorkflows, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)
		assert.True(t, createWorkflows)
	})

	t.Run("returns false when no config file exists", func(t *testing.T) {
		tmpDir := t.TempDir()

		_, _, _, createWorkflows, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, createWorkflows)
	})

	t.Run("returns false when config has no workflows section", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		err := os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644)
		require.NoError(t, err)

		_, _, _, createWorkflows, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, createWorkflows)
	})

	t.Run("returns false when config file is malformed", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(`this is not: [valid yaml`), 0644)
		require.NoError(t, err)

		_, _, _, createWorkflows, _, err := determineInitParameters(tmpDir, log)
		require.NoError(t, err)
		assert.False(t, createWorkflows)
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

	t.Run("init respects existing config workflows.create false", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `terraform_version: "~> 1.13"
workflows:
  create: false
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "111111111111"
      stg: "222222222222"
      prd: "333333333333"
    regions:
      - "eu-central-1"
backend:
  s3:
    bucket_name: "my-terraform-state"
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		initDir = tmpDir
		t.Cleanup(func() {
			initDir = ""
		})

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err = runInit(cmd, []string{})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))

		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		assert.NoDirExists(t, workflowsDir, "workflows should not be created when config has create: false")
	})

	t.Run("init CLI flag overrides config workflows.create", func(t *testing.T) {
		tmpDir := t.TempDir()

		configContent := `terraform_version: "~> 1.13"
workflows:
  create: false
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "111111111111"
      stg: "222222222222"
      prd: "333333333333"
    regions:
      - "eu-central-1"
backend:
  s3:
    bucket_name: "my-terraform-state"
`
		configPath := filepath.Join(tmpDir, ".tfskel.yaml")
		err := os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		initDir = tmpDir
		t.Cleanup(func() {
			initDir = ""
		})

		initWorkflows = true
		t.Cleanup(func() { initWorkflows = false })

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err = runInit(cmd, []string{})
		require.NoError(t, err)

		assert.DirExists(t, filepath.Join(tmpDir, "envs", "dev"))

		workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
		assert.DirExists(t, workflowsDir, "workflows should be created when CLI flag overrides config")
		assert.FileExists(t, filepath.Join(workflowsDir, "lint.yaml"))
	})
}

func TestCheckProjectDrift(t *testing.T) {
	t.Run("no drift when files match config", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		// Generate the project first
		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		// Check for drift — should find none
		drifted := r.checkProjectDrift(baseDir, "1.13.1", []string{"dev"})
		assert.Empty(t, drifted, "no drift expected when files match config")
	})

	t.Run("detects .mise.toml drift when terraform version changes", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		// Generate with version 1.13.1
		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		// Check with version 1.14.0 — .mise.toml and .terraform-version should drift
		drifted := r.checkProjectDrift(baseDir, "1.14.0", []string{"dev"})
		assert.Contains(t, drifted, ".mise.toml")
		assert.Contains(t, drifted, filepath.Join("envs", "dev", ".terraform-version"))
	})

	t.Run("detects .mise.toml drift when tool versions change", func(t *testing.T) {
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)

		r := &initRunner{
			fs:       fs.NewMemoryFileSystem(),
			log:      logger.New(false),
			renderer: renderer,
			tools:    map[string]string{"tflint": "0.50.0"},
		}
		baseDir := "/project"

		// Generate with tflint 0.50.0
		err = r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		// Now check with tflint 0.51.0 — should detect .mise.toml drift
		r.tools = map[string]string{"tflint": "0.51.0"}
		drifted := r.checkProjectDrift(baseDir, "1.13.1", []string{"dev"})
		assert.Contains(t, drifted, ".mise.toml")
	})

	t.Run("no drift for missing files", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/empty-project"

		// Don't generate anything — check against empty directory
		drifted := r.checkProjectDrift(baseDir, "1.13.1", []string{"dev"})
		assert.Empty(t, drifted, "missing files should not be reported as drift")
	})

	t.Run("detects drift across multiple environments", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev", "stg", "prd"}, false)
		require.NoError(t, err)

		// Bump terraform version
		drifted := r.checkProjectDrift(baseDir, "1.14.0", []string{"dev", "stg", "prd"})
		assert.Contains(t, drifted, ".mise.toml")
		assert.Contains(t, drifted, filepath.Join("envs", "dev", ".terraform-version"))
		assert.Contains(t, drifted, filepath.Join("envs", "stg", ".terraform-version"))
		assert.Contains(t, drifted, filepath.Join("envs", "prd", ".terraform-version"))
	})

	t.Run("does not report root config files as drifted when template unchanged", func(t *testing.T) {
		r := newTestInitRunner(t)
		baseDir := "/project"

		err := r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		// Only change tool versions — root config files (.gitignore, etc) should not drift
		r.tools = map[string]string{"tflint": "0.99.0"}
		drifted := r.checkProjectDrift(baseDir, "1.13.1", []string{"dev"})
		assert.NotContains(t, drifted, ".gitignore")
		assert.NotContains(t, drifted, ".pre-commit-config.yaml")
		assert.NotContains(t, drifted, ".tflint.hcl")
		assert.NotContains(t, drifted, "trivy.yaml")
		assert.Contains(t, drifted, ".mise.toml", "only .mise.toml should drift when tool version changes")
	})
}

func TestUpgradeFileDetectsDataDrift(t *testing.T) {
	t.Run("upgrade detects terraform version drift in .mise.toml", func(t *testing.T) {
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)

		r := &initRunner{
			fs:       fs.NewMemoryFileSystem(),
			log:      logger.New(false),
			renderer: renderer,
			upgrade:  true,
		}
		baseDir := "/project"

		// Generate with version 1.13.1
		err = r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		// Read the original .mise.toml
		misePath := filepath.Join(baseDir, ".mise.toml")
		original, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		assert.Contains(t, string(original), `terraform = "1.13.1"`)

		// Now upgrade with new version — should detect data drift and update
		miseData := &templates.Data{
			TerraformVersion: "1.14.0",
			Environments:     []string{"dev"},
		}
		err = r.upgradeFile(misePath, "root/.mise.toml.tmpl", miseData, ".mise.toml")
		require.NoError(t, err)

		upgraded, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		assert.Contains(t, string(upgraded), `terraform = "1.14.0"`, "upgrade should update to new version")
		assert.NotContains(t, string(upgraded), `terraform = "1.13.1"`)
	})

	t.Run("upgrade skips when content matches", func(t *testing.T) {
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)

		r := &initRunner{
			fs:       fs.NewMemoryFileSystem(),
			log:      logger.New(false),
			renderer: renderer,
			upgrade:  true,
		}
		baseDir := "/project"

		err = r.createProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		misePath := filepath.Join(baseDir, ".mise.toml")
		original, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)

		// Upgrade with same data — should skip
		miseData := &templates.Data{
			TerraformVersion: "1.13.1",
			Environments:     []string{"dev"},
		}
		err = r.upgradeFile(misePath, "root/.mise.toml.tmpl", miseData, ".mise.toml")
		require.NoError(t, err)

		after, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		assert.Equal(t, string(original), string(after), "content should not change when versions match")
	})
}

func TestCheckFlagValidation(t *testing.T) {
	t.Run("--check conflicts with --upgrade", func(t *testing.T) {
		tmpDir := t.TempDir()
		initDir = tmpDir
		initCheck = true
		initUpgrade = true
		t.Cleanup(func() {
			initDir = ""
			initCheck = false
			initUpgrade = false
		})

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err := runInit(cmd, []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCheckConflictsWithUpgrade)
	})

	t.Run("--check returns no error when in sync", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Seed config so init and check resolve the same terraform version
		configContent := `terraform_version: "1.13.1"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644))

		initDir = tmpDir
		initCheck = false
		t.Cleanup(func() {
			initDir = ""
			initCheck = false
		})

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err := runInit(cmd, []string{})
		require.NoError(t, err)

		// Now run --check — should pass
		initCheck = true
		err = runInit(cmd, []string{})
		assert.NoError(t, err)
	})

	t.Run("--check returns error when drifted", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Create a config with terraform 1.13
		configContent := `terraform_version: "~> 1.13"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644))

		// Init the project
		initDir = tmpDir
		initCheck = false
		t.Cleanup(func() {
			initDir = ""
			initCheck = false
		})

		cmd := &cobra.Command{}
		cmd.Flags().String("config", "", "config file")

		err := runInit(cmd, []string{})
		require.NoError(t, err)

		// Now change the config to terraform 1.14
		configContent = `terraform_version: "~> 1.14"
provider:
  aws:
    account_mapping:
      dev: "111111111111"
`
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, ".tfskel.yaml"), []byte(configContent), 0644))

		// Run --check — should detect drift
		initCheck = true
		err = runInit(cmd, []string{})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrConfigDrift)
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
