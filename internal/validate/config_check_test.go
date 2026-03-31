package validate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTFFile is a test helper that writes a .tf file with the given HCL content.
func writeTFFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func TestRunConfigCheck(t *testing.T) {
	t.Run("empty directory returns pass with zero files", func(t *testing.T) {
		dir := t.TempDir()

		findings, result, err := RunConfigCheck(&config.Config{}, dir)

		require.NoError(t, err)
		assert.Empty(t, findings)
		assert.Equal(t, StatusPass, result.Status)
		assert.Equal(t, 0, result.Total)
	})

	t.Run("identical version strings has no drift", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "versions.tf", `
terraform {
  required_version = "~> 1.13"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
  }
}
`)

		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 5.80",
				},
			},
		}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Empty(t, findings)
		assert.Equal(t, StatusPass, result.Status)
		assert.Equal(t, 1, result.Total)
	})

	t.Run("semver config vs constraint in tf with same major.minor is in-sync", func(t *testing.T) {
		dir := t.TempDir()
		// Real-world scenario: config has "1.13.1", .tf has "~> 1.13".
		// Same major.minor — should be treated as in-sync.
		writeTFFile(t, dir, "versions.tf", `
terraform {
  required_version = "~> 1.13"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.80"
    }
  }
}
`)

		cfg := &config.Config{
			TerraformVersion: "1.13.1",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "5.80.0",
				},
			},
		}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Empty(t, findings)
		assert.Equal(t, StatusPass, result.Status)
		assert.Equal(t, 1, result.Total)
	})

	t.Run("major terraform version drift produces error finding", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "versions.tf", `
terraform {
  required_version = "~> 0.12"
}
`)

		cfg := &config.Config{
			TerraformVersion: "1.13.1",
		}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Equal(t, StatusFail, result.Status)
		require.NotEmpty(t, findings)

		tf := findings[0]
		assert.Equal(t, CheckConfig, tf.Check)
		assert.Equal(t, "version constraint drift", tf.Message)
		assert.Equal(t, "terraform", tf.Component)
		assert.Equal(t, "version constraint drift", tf.Message)
		assert.NotEmpty(t, tf.Expected)
		assert.Equal(t, "~> 0.12", tf.Actual)
	})

	t.Run("provider version drift produces finding", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "versions.tf", `
terraform {
  required_version = "~> 1.13"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}
`)

		cfg := &config.Config{
			TerraformVersion: "1.13.1",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "5.80.0",
				},
			},
		}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Equal(t, StatusFail, result.Status)

		var awsFindings []Finding
		for _, f := range findings {
			if f.Component == "aws" {
				awsFindings = append(awsFindings, f)
			}
		}
		require.NotEmpty(t, awsFindings)
		assert.Equal(t, "version constraint drift", awsFindings[0].Message)
		assert.Equal(t, "~> 4.0", awsFindings[0].Actual)
	})

	t.Run("multiple tf files across subdirectories", func(t *testing.T) {
		dir := t.TempDir()

		writeTFFile(t, filepath.Join(dir, "envs", "dev"), "versions.tf", `
terraform {
  required_version = "~> 0.12"
}
`)
		writeTFFile(t, filepath.Join(dir, "envs", "prod"), "versions.tf", `
terraform {
  required_version = "~> 1.13"
}
`)

		cfg := &config.Config{
			TerraformVersion: "1.13.1",
		}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Equal(t, 2, result.Total)

		// dev should drift, prod should not.
		driftedFiles := make([]string, 0, len(findings))
		for _, f := range findings {
			driftedFiles = append(driftedFiles, f.Resource)
		}
		assert.Contains(t, driftedFiles, filepath.Join("envs", "dev", "versions.tf"))
	})

	t.Run("file with no version blocks is not counted", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "main.tf", `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
}
`)

		cfg := &config.Config{TerraformVersion: "1.13.1"}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Empty(t, findings)
		assert.Equal(t, StatusPass, result.Status)
		assert.Equal(t, 0, result.Total)
	})

	t.Run("terraform-version file mismatch produces warning", func(t *testing.T) {
		dir := t.TempDir()
		// Create .terraform-version with wrong version.
		envsDir := filepath.Join(dir, "envs", "dev")
		require.NoError(t, os.MkdirAll(envsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(envsDir, ".terraform-version"), []byte("1.12.0\n"), 0o644))

		cfg := &config.Config{TerraformVersion: "~> 1.13"}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Equal(t, StatusFail, result.Status)

		var tvFindings []Finding
		for _, f := range findings {
			if f.Component == "terraform-version-file" {
				tvFindings = append(tvFindings, f)
			}
		}
		require.Len(t, tvFindings, 1)
		assert.Equal(t, "version mismatch", tvFindings[0].Message)
		assert.Equal(t, "1.13", tvFindings[0].Expected)
		assert.Equal(t, "1.12.0", tvFindings[0].Actual)
		assert.Contains(t, tvFindings[0].Resource, ".terraform-version")
	})

	t.Run("terraform-version file matching config produces no finding", func(t *testing.T) {
		dir := t.TempDir()
		envsDir := filepath.Join(dir, "envs", "dev")
		require.NoError(t, os.MkdirAll(envsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(envsDir, ".terraform-version"), []byte("1.13\n"), 0o644))

		cfg := &config.Config{TerraformVersion: "~> 1.13"}

		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Equal(t, StatusPass, result.Status)
		for _, f := range findings {
			assert.NotEqual(t, "terraform-version-file", f.Component, "matching .terraform-version should not produce findings")
		}
	})

	t.Run("multiple terraform-version files across envs", func(t *testing.T) {
		dir := t.TempDir()
		for _, env := range []string{"dev", "stg", "prd"} {
			envsDir := filepath.Join(dir, "envs", env)
			require.NoError(t, os.MkdirAll(envsDir, 0o755))
			v := "1.13"
			if env == "prd" {
				v = "1.12.0" // drift
			}
			require.NoError(t, os.WriteFile(filepath.Join(envsDir, ".terraform-version"), []byte(v+"\n"), 0o644))
		}

		cfg := &config.Config{TerraformVersion: "~> 1.13"}

		findings, _, err := RunConfigCheck(cfg, dir)
		require.NoError(t, err)

		var tvFindings []Finding
		for _, f := range findings {
			if f.Component == "terraform-version-file" {
				tvFindings = append(tvFindings, f)
			}
		}
		assert.Len(t, tvFindings, 1, "only prd should have drift")
		assert.Contains(t, tvFindings[0].Resource, "prd")
	})

	t.Run("empty config flags missing expected version as drift", func(t *testing.T) {
		dir := t.TempDir()
		writeTFFile(t, dir, "versions.tf", `
terraform {
  required_version = "~> 1.13"
}
`)

		// Empty config has no terraform version set.
		// The analyzer treats this as a missing expected version → drift.
		cfg := &config.Config{}
		findings, result, err := RunConfigCheck(cfg, dir)

		require.NoError(t, err)
		assert.Equal(t, StatusFail, result.Status)
		require.NotEmpty(t, findings)
		assert.Equal(t, "terraform", findings[0].Component)
	})
}
