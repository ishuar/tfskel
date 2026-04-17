package bootstrap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/generate"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestRunner creates a Runner backed by MemoryFileSystem for unit tests.
func newTestRunner(t *testing.T) *Runner {
	t.Helper()
	renderer, err := templates.NewRenderer()
	require.NoError(t, err)
	return &Runner{
		fs:       fs.NewMemoryFileSystem(),
		log:      logger.New(false),
		renderer: renderer,
	}
}

// newUpgradeTestRunner creates a Runner in upgrade mode with the given force flag.
func newUpgradeTestRunner(t *testing.T, force bool) *Runner {
	t.Helper()
	r := newTestRunner(t)
	r.upgrade = true
	r.force = force
	return r
}

func TestCreateProjectStructure(t *testing.T) {
	t.Run("create structure with single region", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		for _, filename := range []string{".gitignore", ".pre-commit-config.yaml", ".tflint.hcl", "trivy.yaml", ".tfskel.yaml", ".mise.toml"} {
			filePath := filepath.Join(baseDir, filename)
			assert.True(t, r.fs.FileExists(filePath), "Root config file %s should exist", filename)

			content, err := r.fs.ReadFile(filePath)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "Root config file %s should not be empty", filename)
		}

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
		r := newTestRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}
		regions := []string{"eu-central-1", "us-east-1", "ap-south-1"}

		err := r.CreateProjectStructure(baseDir, "1.10.0", regions, environments, true)
		require.NoError(t, err)

		for _, env := range environments {
			for _, region := range regions {
				regionPath := filepath.Join(baseDir, "envs", env, region)
				assert.True(t, r.fs.DirExists(regionPath), "Region directory %s/%s should exist", env, region)
			}
		}
	})

	t.Run("creates root config files via loop without errors", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		for _, filename := range []string{".pre-commit-config.yaml", ".tflint.hcl", "trivy.yaml"} {
			filePath := filepath.Join(baseDir, filename)
			assert.True(t, r.fs.FileExists(filePath), "File %s from loop should exist", filename)

			content, err := r.fs.ReadFile(filePath)
			require.NoError(t, err)
			assert.NotEmpty(t, content, "File %s should have content", filename)
		}
	})

	t.Run("handles existing root config files gracefully", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"
		environments := []string{"dev", "stg", "prd"}

		existingFile := filepath.Join(baseDir, ".gitignore")
		require.NoError(t, r.fs.WriteFile(existingFile, []byte("# existing content"), 0644))

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(existingFile)
		require.NoError(t, err)
		assert.Equal(t, "# existing content", string(content))

		assert.True(t, r.fs.FileExists(filepath.Join(baseDir, ".pre-commit-config.yaml")))
		assert.True(t, r.fs.FileExists(filepath.Join(baseDir, ".tflint.hcl")))
	})

	t.Run("creates correct terraform version in all environments", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"
		version := "1.9.5"
		environments := []string{"dev", "stg", "prd"}

		err := r.CreateProjectStructure(baseDir, version, []string{"eu-central-1"}, environments, true)
		require.NoError(t, err)

		for _, env := range environments {
			tfVersionPath := filepath.Join(baseDir, "envs", env, ".terraform-version")
			content, err := r.fs.ReadFile(tfVersionPath)
			require.NoError(t, err)
			assert.Contains(t, string(content), version, "Version in %s should match", env)
		}
	})

	t.Run("create structure with custom environments", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"
		customEnvs := []string{"dev", "qa", "uat", "prd"}

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, customEnvs, true)
		require.NoError(t, err)

		for _, env := range customEnvs {
			envPath := filepath.Join(baseDir, "envs", env)
			assert.True(t, r.fs.FileExists(filepath.Join(envPath, ".terraform-version")), "Environment %s should have .terraform-version", env)
			assert.True(t, r.fs.DirExists(filepath.Join(envPath, "eu-central-1")))
		}
	})

	t.Run("creates static workflow files when createWorkflows is true", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, true)
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
		r := newTestRunner(t)
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		workflowsDir := filepath.Join(baseDir, ".github", "workflows")
		assert.False(t, r.fs.DirExists(workflowsDir), ".github/workflows directory should not be created when disabled")
	})

	t.Run("dry-run does not create any files or directories", func(t *testing.T) {
		memFS := fs.NewMemoryFileSystem()
		dryFS := fs.NewDryRunFileSystem(memFS)
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)

		r := &Runner{
			fs:       dryFS,
			log:      logger.New(false),
			renderer: renderer,
			dryRun:   true,
		}
		baseDir := "/project"

		err = r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev", "stg", "prd"}, true)
		require.NoError(t, err)

		assert.False(t, memFS.FileExists(filepath.Join(baseDir, ".gitignore")))
		assert.False(t, memFS.FileExists(filepath.Join(baseDir, ".tfskel.yaml")))
		assert.False(t, memFS.DirExists(filepath.Join(baseDir, "envs", "dev", "eu-central-1")))
		assert.False(t, memFS.FileExists(filepath.Join(baseDir, ".github", "workflows", "lint.yaml")))
	})

	t.Run("does not overwrite existing static workflow files", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		wfDir := filepath.Join(baseDir, ".github", "workflows")
		require.NoError(t, r.fs.MkdirAll(wfDir, 0755))
		existingPath := filepath.Join(wfDir, "lint.yaml")
		require.NoError(t, r.fs.WriteFile(existingPath, []byte("# custom lint"), 0644))

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, true)
		require.NoError(t, err)

		content, readErr := r.fs.ReadFile(existingPath)
		require.NoError(t, readErr)
		assert.Equal(t, "# custom lint", string(content), "existing workflow file should not be overwritten")
	})

	t.Run("creates .mise.toml with correct terraform version", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		misePath := filepath.Join(baseDir, ".mise.toml")
		assert.True(t, r.fs.FileExists(misePath), ".mise.toml should be created")

		content, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, `terraform = "1.13.1"`)
		assert.Contains(t, contentStr, "min_version")
	})

	t.Run(".mise.toml uses latest for all tools", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, ".mise.toml"))
		require.NoError(t, err)
		contentStr := string(content)
		assert.Contains(t, contentStr, `tflint = "latest"`)
		assert.Contains(t, contentStr, `trivy = "latest"`)
		assert.Contains(t, contentStr, `pre-commit = "latest"`)
		assert.Contains(t, contentStr, `awscli = "latest"`)
	})

	t.Run(".mise.toml strips terraform version constraint", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "~> 1.13", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, ".mise.toml"))
		require.NoError(t, err)
		assert.Contains(t, string(content), `terraform = "1.13"`, "constraint should be stripped")
		assert.NotContains(t, string(content), "~>", "constraint operator should not appear in .mise.toml")
	})

	t.Run("does not overwrite existing .mise.toml", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		misePath := filepath.Join(baseDir, ".mise.toml")
		require.NoError(t, r.fs.WriteFile(misePath, []byte("# custom mise config"), 0644))

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(misePath)
		require.NoError(t, err)
		assert.Equal(t, "# custom mise config", string(content), "existing .mise.toml should not be overwritten")
	})

	t.Run(".gitignore created without source marker", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, ".gitignore"))
		require.NoError(t, err)
		assert.NotEmpty(t, string(content), ".gitignore should have content")
		assert.NotContains(t, string(content), "tfskel-source:", ".gitignore should not have a source marker")
	})

	t.Run(".gitignore not overwritten when exists", func(t *testing.T) {
		r := newTestRunner(t)
		baseDir := "/project"

		gitignorePath := filepath.Join(baseDir, ".gitignore")
		require.NoError(t, r.fs.WriteFile(gitignorePath, []byte("# my custom gitignore"), 0644))

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(gitignorePath)
		require.NoError(t, err)
		assert.Equal(t, "# my custom gitignore", string(content), "existing .gitignore should not be overwritten")
	})

	t.Run(".gitignore skipped on upgrade", func(t *testing.T) {
		r := newTestRunner(t)
		r.upgrade = true
		r.force = true
		baseDir := "/project"

		gitignorePath := filepath.Join(baseDir, ".gitignore")
		require.NoError(t, r.fs.WriteFile(gitignorePath, []byte("# my custom gitignore"), 0644))

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(gitignorePath)
		require.NoError(t, err)
		assert.Equal(t, "# my custom gitignore", string(content), ".gitignore should not be touched on upgrade")
	})

	t.Run("skip single file during upgrade", func(t *testing.T) {
		r := newTestRunner(t)
		r.upgrade = true
		r.force = true
		r.skip = map[string]bool{"trivy.yaml": true}
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		require.NoError(t, r.fs.WriteFile(filepath.Join(baseDir, "trivy.yaml"), []byte("# custom trivy"), 0644))

		err = r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(filepath.Join(baseDir, "trivy.yaml"))
		require.NoError(t, err)
		assert.Equal(t, "# custom trivy", string(content), "trivy.yaml should be skipped during upgrade")
	})

	t.Run("skip multiple files during upgrade", func(t *testing.T) {
		r := newTestRunner(t)
		r.upgrade = true
		r.force = true
		r.skip = map[string]bool{"trivy.yaml": true, ".tflint.hcl": true}
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		require.NoError(t, r.fs.WriteFile(filepath.Join(baseDir, "trivy.yaml"), []byte("# custom trivy"), 0644))
		require.NoError(t, r.fs.WriteFile(filepath.Join(baseDir, ".tflint.hcl"), []byte("# custom tflint"), 0644))

		err = r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		trivyContent, _ := r.fs.ReadFile(filepath.Join(baseDir, "trivy.yaml"))
		assert.Equal(t, "# custom trivy", string(trivyContent))
		tflintContent, _ := r.fs.ReadFile(filepath.Join(baseDir, ".tflint.hcl"))
		assert.Equal(t, "# custom tflint", string(tflintContent))

		assert.True(t, r.fs.FileExists(filepath.Join(baseDir, ".pre-commit-config.yaml")))
	})

	t.Run("skip has no effect on first-time creation", func(t *testing.T) {
		r := newTestRunner(t)
		r.skip = map[string]bool{"trivy.yaml": true}
		baseDir := "/project"

		err := r.CreateProjectStructure(baseDir, "1.13.1", []string{"eu-central-1"}, []string{"dev"}, false)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(filepath.Join(baseDir, "trivy.yaml")), "skip should not prevent first-time creation")
	})
}

func TestCreateFileFromTemplate(t *testing.T) {
	t.Run("create file from template with nil data", func(t *testing.T) {
		r := newTestRunner(t)
		targetPath := "/project/test/file.txt"

		err := r.createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(targetPath))
		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.NotEmpty(t, string(content))
	})

	t.Run("create file from template with data", func(t *testing.T) {
		r := newTestRunner(t)
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
		r := newTestRunner(t)
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
		r := newTestRunner(t)
		targetPath := "/project/deep/nested/path/file.txt"

		err := r.createFileFromTemplate(targetPath, "root/.gitignore.tmpl", nil)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(targetPath))
	})

	t.Run("handles multiple file creations in sequence", func(t *testing.T) {
		r := newTestRunner(t)

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
		r := newTestRunner(t)
		configPath := "/project/.tfskel.yaml"

		err := r.CreateDefaultConfig(configPath)
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
		r := newTestRunner(t)
		configPath := "/project/.tfskel.yaml"

		require.NoError(t, r.fs.WriteFile(configPath, []byte("existing"), 0644))

		err := r.CreateDefaultConfig(configPath)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, "existing", string(content))
	})
}

func TestUpgradeFile(t *testing.T) {
	const templateName = "root/.gitignore.tmpl"

	t.Run("no marker, no force — skips", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"
		originalContent := "# my custom gitignore\n*.log\n"

		require.NoError(t, r.fs.WriteFile(targetPath, []byte(originalContent), 0644))

		err := r.upgradeFile(targetPath, templateName, nil, ".gitignore")
		require.NoError(t, err)

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(content), "content should be unchanged when no marker and no force")
	})

	t.Run("no marker, force — overwrites", func(t *testing.T) {
		r := newUpgradeTestRunner(t, true)
		targetPath := "/project/.gitignore"
		originalContent := "# my custom gitignore\n*.log\n"

		require.NoError(t, r.fs.WriteFile(targetPath, []byte(originalContent), 0644))

		err := r.upgradeFile(targetPath, templateName, nil, ".gitignore")
		require.NoError(t, err)

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.NotEqual(t, originalContent, string(content), "content should be overwritten with force")
		assert.Contains(t, string(content), "tfskel-source:", "re-rendered content should have source marker")
	})

	t.Run("invalid marker, no force — returns error", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"
		invalidContent := "# tfskel-source: {not valid json!}\n*.log\n"

		require.NoError(t, r.fs.WriteFile(targetPath, []byte(invalidContent), 0644))

		err := r.upgradeFile(targetPath, templateName, nil, ".gitignore")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid source marker")
	})

	t.Run("invalid marker, force — overwrites", func(t *testing.T) {
		r := newUpgradeTestRunner(t, true)
		targetPath := "/project/.gitignore"
		invalidContent := "# tfskel-source: {not valid json!}\n*.log\n"

		require.NoError(t, r.fs.WriteFile(targetPath, []byte(invalidContent), 0644))

		err := r.upgradeFile(targetPath, templateName, nil, ".gitignore")
		require.NoError(t, err)

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "tfskel-source:", "re-rendered content should have valid marker")
	})

	t.Run("valid marker — delegates to upgradeFromMarker", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"

		r.upgrade = false
		require.NoError(t, r.createFileFromTemplate(targetPath, templateName, nil))
		r.upgrade = true

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "tfskel-source:", "file should have marker from initial creation")

		err = r.upgradeFile(targetPath, templateName, nil, ".gitignore")
		require.NoError(t, err)
	})
}

func TestUpgradeFromMarker(t *testing.T) {
	const templateName = "root/.gitignore.tmpl"

	t.Run("template name mismatch — skips", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"

		r.upgrade = false
		require.NoError(t, r.createFileFromTemplate(targetPath, templateName, nil))
		r.upgrade = true

		originalContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)

		marker, err := generate.ExtractSourceMarker(string(originalContent))
		require.NoError(t, err)

		err = r.upgradeFromMarker(targetPath, "root/.tflint.hcl.tmpl", nil, ".gitignore", marker, originalContent, "Upgrading")
		require.NoError(t, err)

		afterContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "content should be unchanged on template mismatch")
	})

	t.Run("content unchanged — skips", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"

		r.upgrade = false
		require.NoError(t, r.createFileFromTemplate(targetPath, templateName, nil))
		r.upgrade = true

		originalContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)

		marker, err := generate.ExtractSourceMarker(string(originalContent))
		require.NoError(t, err)

		err = r.upgradeFromMarker(targetPath, templateName, nil, ".gitignore", marker, originalContent, "Upgrading")
		require.NoError(t, err)

		afterContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, string(originalContent), string(afterContent), "content should be unchanged when already up to date")
	})

	t.Run("template hash changed — upgrades", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"

		r.upgrade = false
		require.NoError(t, r.createFileFromTemplate(targetPath, templateName, nil))
		r.upgrade = true

		originalContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)

		realHash := r.renderer.GetTemplateHash(templateName)
		modifiedContent := strings.Replace(string(originalContent), realHash, "fakehash000", 1)
		require.NoError(t, r.fs.WriteFile(targetPath, []byte(modifiedContent), 0644))

		marker, err := generate.ExtractSourceMarker(modifiedContent)
		require.NoError(t, err)
		assert.Equal(t, "fakehash000", marker.Hash, "marker should have fake hash")

		err = r.upgradeFromMarker(targetPath, templateName, nil, ".gitignore", marker, []byte(modifiedContent), "Upgrading")
		require.NoError(t, err)

		afterContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.NotEqual(t, modifiedContent, string(afterContent), "content should be updated")
		assert.Contains(t, string(afterContent), realHash, "updated content should have current hash")
	})

	t.Run("content drift — upgrades", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"

		r.upgrade = false
		require.NoError(t, r.createFileFromTemplate(targetPath, templateName, nil))
		r.upgrade = true

		originalContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)

		driftedContent := string(originalContent) + "\n# manually added line\n"
		require.NoError(t, r.fs.WriteFile(targetPath, []byte(driftedContent), 0644))

		marker, err := generate.ExtractSourceMarker(driftedContent)
		require.NoError(t, err)

		err = r.upgradeFromMarker(targetPath, templateName, nil, ".gitignore", marker, []byte(driftedContent), "Upgrading")
		require.NoError(t, err)

		afterContent, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.NotContains(t, string(afterContent), "manually added line", "drifted content should be replaced")
	})
}

func TestCreateFileFromTemplate_UpgradePath(t *testing.T) {
	const templateName = "root/.gitignore.tmpl"

	t.Run("upgrade=true, file exists without marker — skips", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"
		originalContent := "# custom content\n"

		require.NoError(t, r.fs.WriteFile(targetPath, []byte(originalContent), 0644))

		err := r.createFileFromTemplate(targetPath, templateName, nil)
		require.NoError(t, err)

		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, originalContent, string(content), "file without marker should be skipped on upgrade")
	})

	t.Run("upgrade=true, file does not exist — creates normally", func(t *testing.T) {
		r := newUpgradeTestRunner(t, false)
		targetPath := "/project/.gitignore"

		err := r.createFileFromTemplate(targetPath, templateName, nil)
		require.NoError(t, err)

		assert.True(t, r.fs.FileExists(targetPath))
		content, err := r.fs.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "tfskel-source:", "new file should have source marker")
	})
}
