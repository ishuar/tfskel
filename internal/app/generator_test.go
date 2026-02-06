package app

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
)

func TestNewGenerator(t *testing.T) {
	t.Run("create new generator", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)

		gen := NewGenerator(cfg, filesystem, log)

		assert.NotNil(t, gen)
		assert.Equal(t, cfg, gen.config)
		assert.Equal(t, filesystem, gen.fs)
		assert.Equal(t, log, gen.log)
	})
}

func TestGenerator_generateFiles(t *testing.T) {
	t.Run("generate all files from templates", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-terraform-state-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)
		// Check that templates are created (only non-root level templates)
		expectedFiles := []string{
			"versions.tf",
			"backend.tf",
		}

		for _, file := range expectedFiles {
			filePath := filepath.Join(appPath, file)
			assert.True(t, filesystem.FileExists(filePath), "expected file %s to exist", file)

			content, readErr := filesystem.ReadFile(filePath)
			assert.NoError(t, readErr)
			assert.NotEmpty(t, content, "expected file %s to have content", file)
		}
	})

	t.Run("skip existing files", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-terraform-state-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)
		existingContent := []byte("existing content")
		_ = filesystem.WriteFile(filepath.Join(appPath, "variables.tf"), existingContent, 0644)

		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)

		content, readErr := filesystem.ReadFile(filepath.Join(appPath, "variables.tf"))
		assert.NoError(t, readErr)
		assert.Equal(t, existingContent, content)
	})
}

func TestGenerator_Run_Integration(t *testing.T) {
	t.Run("full generation workflow", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-terraform-state-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)

		gen := NewGenerator(cfg, filesystem, log)
		err := gen.Run("dev", "eu-central-1", "testapp")

		assert.NoError(t, err)

		appPath := "envs/dev/eu-central-1/testapp"
		assert.True(t, filesystem.DirExists(appPath))

		// Check that templates exist (only non-root level templates)
		expectedFiles := []string{
			"versions.tf",
			"backend.tf",
		}

		for _, file := range expectedFiles {
			filePath := filepath.Join(appPath, file)
			assert.True(t, filesystem.FileExists(filePath), "expected file %s to exist", file)
		}
	})

	t.Run("generation with custom templates", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"stg": "987654321098",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-terraform-state-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)

		gen := NewGenerator(cfg, filesystem, log)
		err := gen.Run("stg", "us-west-2", "myapp")

		assert.NoError(t, err)

		appPath := "envs/stg/us-west-2/myapp"
		assert.True(t, filesystem.DirExists(appPath))
	})
}

func TestGenerator_renderBucketName(t *testing.T) {
	t.Run("render simple bucket name template with Env", func(t *testing.T) {
		cfg := &config.Config{}
		gen := NewGenerator(cfg, fs.NewMemoryFileSystem(), logger.New(false))

		data := &templates.Data{
			Env:    "dev",
			Region: "us-east-1",
			AppDir: "myapp",
		}

		result, err := gen.renderBucketName("{{.Env}}-terraform-state", data)
		assert.NoError(t, err)
		assert.Equal(t, "dev-terraform-state", result)
	})

	t.Run("render bucket name template with multiple variables", func(t *testing.T) {
		cfg := &config.Config{}
		gen := NewGenerator(cfg, fs.NewMemoryFileSystem(), logger.New(false))

		data := &templates.Data{
			Env:    "prd",
			Region: "eu-central-1",
			AppDir: "webapp",
		}

		result, err := gen.renderBucketName("{{.AppDir}}-{{.Env}}-{{.Region}}-tfstate", data)
		assert.NoError(t, err)
		assert.Equal(t, "webapp-prd-eu-central-1-tfstate", result)
	})

	t.Run("render bucket name without template syntax", func(t *testing.T) {
		cfg := &config.Config{}
		gen := NewGenerator(cfg, fs.NewMemoryFileSystem(), logger.New(false))

		data := &templates.Data{
			Env:    "dev",
			Region: "us-east-1",
		}

		result, err := gen.renderBucketName("static-bucket-name", data)
		assert.NoError(t, err)
		assert.Equal(t, "static-bucket-name", result)
	})

	t.Run("error on invalid template syntax", func(t *testing.T) {
		cfg := &config.Config{}
		gen := NewGenerator(cfg, fs.NewMemoryFileSystem(), logger.New(false))

		data := &templates.Data{Env: "dev"}

		_, err := gen.renderBucketName("{{.Env", data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse bucket_name template")
	})
}

func TestGenerator_shouldUpdateBackend(t *testing.T) {
	t.Run("returns true when bucket name differs", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		backendContent := `## tfskel-metadata: {"bucket": "old-bucket-name"}
terraform {
  backend "s3" {
    bucket = "old-bucket-name"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}`
		_ = filesystem.WriteFile("backend.tf", []byte(backendContent), 0644)

		needsUpdate, err := gen.shouldUpdateBackend("backend.tf", "new-bucket-name")
		assert.NoError(t, err)
		assert.True(t, needsUpdate)
	})

	t.Run("returns false when bucket name matches", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		backendContent := `## tfskel-metadata: {"bucket": "same-bucket-name"}
terraform {
  backend "s3" {
    bucket = "same-bucket-name"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}`
		_ = filesystem.WriteFile("backend.tf", []byte(backendContent), 0644)

		needsUpdate, err := gen.shouldUpdateBackend("backend.tf", "same-bucket-name")
		assert.NoError(t, err)
		assert.False(t, needsUpdate)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		needsUpdate, err := gen.shouldUpdateBackend("nonexistent.tf", "bucket-name")
		assert.Error(t, err)
		assert.False(t, needsUpdate)
	})

	t.Run("returns true when metadata not found (needs regeneration)", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		// Old file without metadata
		backendContent := `terraform {
  backend "s3" {
    bucket = "old-style-bucket"
  }
}`
		_ = filesystem.WriteFile("backend.tf", []byte(backendContent), 0644)

		needsUpdate, err := gen.shouldUpdateBackend("backend.tf", "old-style-bucket")
		assert.NoError(t, err)
		assert.True(t, needsUpdate) // Should regenerate to add metadata
	})
}

func TestGenerator_updateBackendFile(t *testing.T) {
	t.Run("successfully updates backend file", func(t *testing.T) {
		cfg := &config.Config{
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{"dev": "123456789012"},
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		data := &templates.Data{
			Env:          "dev",
			Region:       "us-east-1",
			AppDir:       "myapp",
			AccountID:    "123456789012",
			S3BucketName: "new-bucket-name",
		}

		err = gen.updateBackendFile("backend.tf", data)
		assert.NoError(t, err)

		content, readErr := filesystem.ReadFile("backend.tf")
		assert.NoError(t, readErr)
		assert.Contains(t, string(content), `bucket              = "new-bucket-name"`)
	})
}

func TestGenerator_BucketNameCascadeUpdate(t *testing.T) {
	t.Run("cascades bucket name changes to existing backend.tf", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"prd": "210987654321",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "{{.Env}}-terraform-state",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Create app path and existing backend.tf with old bucket name
		appPath := "envs/prd/eu-central-1/webapp"
		_ = filesystem.MkdirAll(appPath, 0755)
		oldBackendContent := `terraform {
  backend "s3" {
    bucket = "old-bucket-name"
    key    = "terraform.tfstate"
  }
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "backend.tf"), []byte(oldBackendContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles which should cascade the update
		err = gen.generateFiles(appPath, "prd", "eu-central-1", "webapp")
		assert.NoError(t, err)

		// Verify backend.tf was updated with rendered template
		content, readErr := filesystem.ReadFile(filepath.Join(appPath, "backend.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(content), `bucket              = "prd-terraform-state"`)
		assert.NotContains(t, string(content), "old-bucket-name")
	})

	t.Run("does not update backend.tf when bucket name matches", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "dev-terraform-state",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Create app path and existing backend.tf with matching bucket name AND metadata
		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)
		matchingBackendContent := `## This file is auto generated by tfskel
## Verify the bucket name & make sure it exists in your AWS account.
## Verify other backend configuration as per your requirements before running 'terraform init'
## docs ref: https://developer.hashicorp.com/terraform/language/backend/s3
## tfskel-metadata: {"bucket": "dev-terraform-state"}

terraform {
  backend "s3" {
    bucket              = "dev-terraform-state"
    key                 = "myapp-us-east-1-dev/terraform.tfstate"
    region              = "us-east-1"
    encrypt             = true
    use_lockfile        = true
    allowed_account_ids = ["123456789012"]
  }
}
`
		_ = filesystem.WriteFile(filepath.Join(appPath, "backend.tf"), []byte(matchingBackendContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles which should NOT update the file
		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)

		// Verify backend.tf content remains the same (includes specific formatting)
		content, readErr := filesystem.ReadFile(filepath.Join(appPath, "backend.tf"))
		assert.NoError(t, readErr)
		assert.Equal(t, matchingBackendContent, string(content))
	})
}

func TestGenerator_shouldUpdateVersions(t *testing.T) {
	t.Run("returns true when terraform version differs", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		versionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.12", "aws_provider_ver": "~> 5.0"}
terraform {
  required_version = "~> 1.12"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}`
		_ = filesystem.WriteFile("versions.tf", []byte(versionsContent), 0644)

		data := &templates.Data{
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 5.0",
			DefaultTags:        make(map[string]string),
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.True(t, needsUpdate)
		assert.Contains(t, changes[0], "tf_ver changed: ~> 1.12 -> ~> 1.13")
	})

	t.Run("returns true when AWS provider version differs", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		versionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 5.0"}
terraform {
  required_version = "~> 1.13"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}`
		_ = filesystem.WriteFile("versions.tf", []byte(versionsContent), 0644)

		data := &templates.Data{
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 6.0",
			DefaultTags:        make(map[string]string),
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.True(t, needsUpdate)
		assert.Contains(t, changes[0], "aws_provider_ver changed: ~> 5.0 -> ~> 6.0")
	})

	t.Run("returns true when default_tags differ", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		versionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {"managed_by": "terraform"}
provider "aws" {
  region = basename(dirname(path.cwd))

  default_tags {
    tags = {
      managed_by = "terraform"
      env = "dev"
      app = "myapp"
    }
  }
}`
		_ = filesystem.WriteFile("versions.tf", []byte(versionsContent), 0644)

		data := &templates.Data{
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 6.0",
			DefaultTags: map[string]string{
				"managed_by": "terraform",
				"team":       "platform",
			},
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.True(t, needsUpdate)
		assert.Contains(t, changes[0], "team added: platform")
	})

	t.Run("returns false when all values match", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		versionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
terraform {
  required_version = "~> 1.13"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}`
		_ = filesystem.WriteFile("versions.tf", []byte(versionsContent), 0644)

		data := &templates.Data{
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 6.0",
			DefaultTags:        make(map[string]string),
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.False(t, needsUpdate)
		assert.Empty(t, changes)
	})

	t.Run("returns error when file does not exist", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		data := &templates.Data{
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 6.0",
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("nonexistent.tf", data)
		assert.Error(t, err)
		assert.False(t, needsUpdate)
		assert.Nil(t, changes)
	})

	t.Run("returns true when metadata not found (needs initialization)", func(t *testing.T) {
		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		// Old file without metadata
		versionsContent := `terraform {
  required_version = "~> 1.13"
}`
		_ = filesystem.WriteFile("versions.tf", []byte(versionsContent), 0644)

		data := &templates.Data{
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 6.0",
			DefaultTags:        make(map[string]string),
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.True(t, needsUpdate) // Should regenerate to add metadata
		assert.Contains(t, changes[0], "metadata initialization")
	})
}

func TestGenerator_updateVersionsFile(t *testing.T) {
	t.Run("successfully updates versions file", func(t *testing.T) {
		cfg := &config.Config{
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{"dev": "123456789012"},
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		data := &templates.Data{
			Env:                "dev",
			TerraformVersion:   "~> 1.13",
			AWSProviderVersion: "~> 6.0",
			DefaultTags: map[string]string{
				"managed_by": "terraform",
			},
			AppDir: "myapp",
		}

		err = gen.updateVersionsFile("versions.tf", data)
		assert.NoError(t, err)

		content, readErr := filesystem.ReadFile("versions.tf")
		assert.NoError(t, readErr)
		assert.Contains(t, string(content), `required_version = "~> 1.13"`)
		assert.Contains(t, string(content), `version = "~> 6.0"`)
	})
}

func TestGenerator_VersionsCascadeUpdate(t *testing.T) {
	t.Run("cascades terraform version changes to existing versions.tf", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.14",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"prd": "210987654321",
					},
					DefaultTags: map[string]string{
						"managed_by": "terraform",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Create app path and existing versions.tf with old version
		appPath := "envs/prd/eu-central-1/webapp"
		_ = filesystem.MkdirAll(appPath, 0755)
		oldVersionsContent := `terraform {
  required_version = "~> 1.12"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "versions.tf"), []byte(oldVersionsContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles which should cascade the update
		err = gen.generateFiles(appPath, "prd", "eu-central-1", "webapp")
		assert.NoError(t, err)

		// Verify versions.tf was updated with new terraform version
		content, readErr := filesystem.ReadFile(filepath.Join(appPath, "versions.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(content), `required_version = "~> 1.14"`)
		assert.NotContains(t, string(content), "~> 1.12")
	})

	t.Run("cascades AWS provider version changes to existing versions.tf", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 7.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Create app path and existing versions.tf with old provider version
		appPath := "envs/dev/us-east-1/api"
		_ = filesystem.MkdirAll(appPath, 0755)
		oldVersionsContent := `terraform {
  required_version = "~> 1.13"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "versions.tf"), []byte(oldVersionsContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles which should cascade the update
		err = gen.generateFiles(appPath, "dev", "us-east-1", "api")
		assert.NoError(t, err)

		// Verify versions.tf was updated with new provider version
		content, readErr := filesystem.ReadFile(filepath.Join(appPath, "versions.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(content), `version = "~> 7.0"`)
		assert.NotContains(t, string(content), "~> 6.0")
	})

	t.Run("does not update versions.tf when values match", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
					DefaultTags: map[string]string{
						"managed_by": "terraform",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Create app path and existing versions.tf with matching values AND metadata
		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)
		matchingVersionsContent := `## Terraform providers and required versions
## This file is auto generated by tfskel
## DO NOT REMOVE the tfskel-metadata & tfskel-tags comments for management via tfskel
## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {"managed_by": "terraform", }

terraform {
  required_version = "~> 1.13"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = basename(dirname(path.cwd))

  default_tags {
    tags = {
      managed_by = "terraform"
      env = "dev"
      app = "myapp"
    }
  }
}
`
		_ = filesystem.WriteFile(filepath.Join(appPath, "versions.tf"), []byte(matchingVersionsContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles which should NOT update the file
		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)

		// Verify versions.tf content remains the same
		content, readErr := filesystem.ReadFile(filepath.Join(appPath, "versions.tf"))
		assert.NoError(t, readErr)
		assert.Equal(t, matchingVersionsContent, string(content))
	})
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		metadataKey string
		expected    map[string]string
		expectError bool
	}{
		{
			name: "extract backend metadata",
			content: `## This file is auto generated by tfskel
## tfskel-metadata: {"bucket": "my-terraform-state-bucket"}

terraform {
  backend "s3" {
    bucket = "my-terraform-state-bucket"
  }
}`,
			metadataKey: "metadata",
			expected:    map[string]string{"bucket": "my-terraform-state-bucket"},
			expectError: false,
		},
		{
			name: "extract versions metadata",
			content: `## This file is auto generated by tfskel
## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {"managed_by": "terraform", "team": "platform"}

terraform {
  required_version = "~> 1.13"
}`,
			metadataKey: "metadata",
			expected:    map[string]string{"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"},
			expectError: false,
		},
		{
			name: "extract tags metadata",
			content: `## This file is auto generated by tfskel
## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {"managed_by": "terraform", "team": "platform"}

terraform {
  required_version = "~> 1.13"
}`,
			metadataKey: "tags",
			expected:    map[string]string{"managed_by": "terraform", "team": "platform"},
			expectError: false,
		},
		{
			name:        "metadata not found",
			content:     `## This is a regular comment\nterraform {}`,
			metadataKey: "metadata",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "invalid JSON",
			content:     `## tfskel-metadata: {invalid json}`,
			metadataKey: "metadata",
			expected:    nil,
			expectError: true,
		},
		{
			name:        "empty metadata",
			content:     `## tfskel-metadata: {}`,
			metadataKey: "metadata",
			expected:    map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractMetadata(tt.content, tt.metadataKey)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestBuildBackendMetadata(t *testing.T) {
	tests := []struct {
		name       string
		bucketName string
		expected   map[string]string
	}{
		{
			name:       "simple bucket name",
			bucketName: "my-bucket",
			expected:   map[string]string{"bucket": "my-bucket"},
		},
		{
			name:       "templated bucket name",
			bucketName: "prd-eu-central-1-tfstate",
			expected:   map[string]string{"bucket": "prd-eu-central-1-tfstate"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildBackendMetadata(tt.bucketName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBuildVersionsMetadata(t *testing.T) {
	tests := []struct {
		name            string
		tfVersion       string
		providerVersion string
		expected        map[string]string
	}{
		{
			name:            "standard versions",
			tfVersion:       "~> 1.13",
			providerVersion: "~> 6.0",
			expected:        map[string]string{"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"},
		},
		{
			name:            "different versions",
			tfVersion:       "~> 1.14",
			providerVersion: "~> 7.0",
			expected:        map[string]string{"tf_ver": "~> 1.14", "aws_provider_ver": "~> 7.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildVersionsMetadata(tt.tfVersion, tt.providerVersion)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompareMetadata(t *testing.T) {
	tests := []struct {
		name           string
		fileMetadata   map[string]string
		configMetadata map[string]string
		expectChanges  bool
		expectedMsgs   []string
	}{
		{
			name:           "no changes",
			fileMetadata:   map[string]string{"bucket": "my-bucket"},
			configMetadata: map[string]string{"bucket": "my-bucket"},
			expectChanges:  false,
			expectedMsgs:   []string{},
		},
		{
			name:           "value changed",
			fileMetadata:   map[string]string{"bucket": "old-bucket"},
			configMetadata: map[string]string{"bucket": "new-bucket"},
			expectChanges:  true,
			expectedMsgs:   []string{"bucket changed: old-bucket -> new-bucket"},
		},
		{
			name:           "key added",
			fileMetadata:   map[string]string{},
			configMetadata: map[string]string{"bucket": "my-bucket"},
			expectChanges:  true,
			expectedMsgs:   []string{"bucket added: my-bucket"},
		},
		{
			name:           "key removed",
			fileMetadata:   map[string]string{"bucket": "my-bucket"},
			configMetadata: map[string]string{},
			expectChanges:  true,
			expectedMsgs:   []string{"bucket removed (was: my-bucket)"},
		},
		{
			name:           "multiple changes",
			fileMetadata:   map[string]string{"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"},
			configMetadata: map[string]string{"tf_ver": "~> 1.14", "aws_provider_ver": "~> 6.0"},
			expectChanges:  true,
			expectedMsgs:   []string{"tf_ver changed: ~> 1.13 -> ~> 1.14"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasChanges, changes := compareMetadata(tt.fileMetadata, tt.configMetadata)

			assert.Equal(t, tt.expectChanges, hasChanges)
			if tt.expectChanges {
				assert.Len(t, changes, len(tt.expectedMsgs))
				for _, expectedMsg := range tt.expectedMsgs {
					assert.Contains(t, changes, expectedMsg)
				}
			}
		})
	}
}

func TestCompareTags(t *testing.T) {
	tests := []struct {
		name          string
		fileTags      map[string]string
		configTags    map[string]string
		expectChanges bool
		expectedMsgs  []string
	}{
		{
			name:          "no changes",
			fileTags:      map[string]string{"managed_by": "terraform", "team": "platform"},
			configTags:    map[string]string{"managed_by": "terraform", "team": "platform"},
			expectChanges: false,
			expectedMsgs:  []string{},
		},
		{
			name:          "tag added",
			fileTags:      map[string]string{"managed_by": "terraform"},
			configTags:    map[string]string{"managed_by": "terraform", "team": "platform"},
			expectChanges: true,
			expectedMsgs:  []string{"team added: platform"},
		},
		{
			name:          "tag removed",
			fileTags:      map[string]string{"managed_by": "terraform", "team": "platform"},
			configTags:    map[string]string{"managed_by": "terraform"},
			expectChanges: true,
			expectedMsgs:  []string{"team removed (was: platform)"},
		},
		{
			name:          "tag value changed",
			fileTags:      map[string]string{"managed_by": "terraform", "team": "platform"},
			configTags:    map[string]string{"managed_by": "terraform", "team": "devops"},
			expectChanges: true,
			expectedMsgs:  []string{"team changed: platform -> devops"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasChanges, changes := compareTags(tt.fileTags, tt.configTags)

			assert.Equal(t, tt.expectChanges, hasChanges)
			if tt.expectChanges {
				assert.Len(t, changes, len(tt.expectedMsgs))
				for _, expectedMsg := range tt.expectedMsgs {
					assert.Contains(t, changes, expectedMsg)
				}
			}
		})
	}
}

// Test for simultaneous updates - CRITICAL edge case
func TestGenerator_SimultaneousBackendAndVersionsUpdate(t *testing.T) {
	t.Run("updates BOTH backend and versions when both are outdated", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16", // New version
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 7.0", // New provider version
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "new-bucket-name", // New bucket name
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Create app path with OLD backend and OLD versions
		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		// Old backend.tf with old bucket name
		oldBackendContent := `## tfskel-metadata: {"bucket": "old-bucket-name"}
terraform {
  backend "s3" {
    bucket = "old-bucket-name"
    key    = "terraform.tfstate"
  }
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "backend.tf"), []byte(oldBackendContent), 0644)

		// Old versions.tf with old versions
		oldVersionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
terraform {
  required_version = "~> 1.13"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "versions.tf"), []byte(oldVersionsContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles which should update BOTH files
		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)

		// Verify backend.tf was updated
		backendContent, readErr := filesystem.ReadFile(filepath.Join(appPath, "backend.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(backendContent), `bucket              = "new-bucket-name"`)
		assert.NotContains(t, string(backendContent), "old-bucket-name")
		assert.Contains(t, string(backendContent), `## tfskel-metadata:`)

		// Verify versions.tf was updated
		versionsContent, readErr := filesystem.ReadFile(filepath.Join(appPath, "versions.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(versionsContent), `required_version = "~> 1.16"`)
		assert.Contains(t, string(versionsContent), `version = "~> 7.0"`)
		assert.NotContains(t, string(versionsContent), "~> 1.13")
		assert.NotContains(t, string(versionsContent), "~> 6.0")
		assert.Contains(t, string(versionsContent), `## tfskel-metadata:`)
	})
}

// Test for error handling in metadata extraction
func TestGenerator_ErrorHandlingInCascadeUpdates(t *testing.T) {
	t.Run("regenerates backend.tf when metadata is corrupted", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		// Create backend.tf with corrupted JSON metadata
		corruptedBackendContent := `## tfskel-metadata: {invalid json}}
terraform {
  backend "s3" {
    bucket = "old-bucket"
  }
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "backend.tf"), []byte(corruptedBackendContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles - should succeed by regenerating the file
		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err, "Should regenerate file with corrupted metadata")

		// Verify backend.tf was regenerated with correct metadata
		backendContent, readErr := filesystem.ReadFile(filepath.Join(appPath, "backend.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(backendContent), `bucket              = "my-bucket"`)

		// Verify metadata is now valid
		metadata, metaErr := extractMetadata(string(backendContent), "metadata")
		assert.NoError(t, metaErr, "Regenerated file should have valid metadata")
		assert.Equal(t, "my-bucket", metadata["bucket"])
	})

	t.Run("regenerates versions.tf when metadata is corrupted", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		// Create versions.tf with corrupted JSON metadata
		corruptedVersionsContent := `## tfskel-metadata: {invalid json}}
terraform {
  required_version = "~> 1.13"
}`
		_ = filesystem.WriteFile(filepath.Join(appPath, "versions.tf"), []byte(corruptedVersionsContent), 0644)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Run generateFiles - should succeed by regenerating the file
		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err, "Should regenerate file with corrupted metadata")

		// Verify versions.tf was regenerated with correct metadata
		versionsContent, readErr := filesystem.ReadFile(filepath.Join(appPath, "versions.tf"))
		assert.NoError(t, readErr)
		assert.Contains(t, string(versionsContent), `required_version = "~> 1.16"`)

		// Verify metadata is now valid
		metadata, metaErr := extractMetadata(string(versionsContent), "metadata")
		assert.NoError(t, metaErr, "Regenerated file should have valid metadata")
		assert.Equal(t, "~> 1.16", metadata["tf_ver"])
		assert.Equal(t, "~> 6.0", metadata["aws_provider_ver"])
	})

	t.Run("returns error when backend.tf file cannot be read", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "my-bucket",
				},
			},
		}

		// Use a custom filesystem that fails on ReadFile
		filesystem := &FailingReadFileSystem{
			MemoryFileSystem: fs.NewMemoryFileSystem(),
			failPath:         "envs/dev/us-east-1/myapp/backend.tf",
		}
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		// Initialize renderer
		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		// Create backend.tf that will fail to read
		_ = filesystem.WriteFile(filepath.Join(appPath, "backend.tf"), []byte("content"), 0644)

		// Run generateFiles - should return error when trying to read backend.tf
		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "failed to check backend.tf for updates")
		}
	})
}

// FailingReadFileSystem wraps MemoryFileSystem and fails on specific file reads
type FailingReadFileSystem struct {
	*fs.MemoryFileSystem

	failPath string
}

func (f *FailingReadFileSystem) ReadFile(path string) ([]byte, error) {
	if path == f.failPath {
		return nil, fmt.Errorf("simulated read error for %s", path)
	}
	return f.MemoryFileSystem.ReadFile(path)
}

// Test to ensure metadata is always present in generated files
func TestGenerator_MetadataPresenceValidation(t *testing.T) {
	t.Run("generated backend.tf contains valid metadata", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "test-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)

		// Verify backend.tf has valid metadata that can be parsed
		backendContent, readErr := filesystem.ReadFile(filepath.Join(appPath, "backend.tf"))
		assert.NoError(t, readErr)

		metadata, err := extractMetadata(string(backendContent), "metadata")
		assert.NoError(t, err, "backend.tf should contain valid parseable metadata")
		assert.Equal(t, "test-bucket", metadata["bucket"])
	})

	t.Run("generated versions.tf contains valid metadata", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.16",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "test-bucket",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/us-east-1/myapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "us-east-1", "myapp")
		assert.NoError(t, err)

		// Verify versions.tf has valid metadata that can be parsed
		versionsContent, readErr := filesystem.ReadFile(filepath.Join(appPath, "versions.tf"))
		assert.NoError(t, readErr)

		metadata, err := extractMetadata(string(versionsContent), "metadata")
		assert.NoError(t, err, "versions.tf should contain valid parseable metadata")
		assert.Equal(t, "~> 1.16", metadata["tf_ver"])
		assert.Equal(t, "~> 6.0", metadata["aws_provider_ver"])
	})
}
