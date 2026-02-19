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

func TestGenerator_prepareTemplateData_BucketNameRendering(t *testing.T) {
	t.Run("render simple bucket name template with Env", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{
						"dev": "123456789012",
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

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		data, err := gen.prepareTemplateData("dev", "us-east-1", "myapp")
		assert.NoError(t, err)
		assert.Equal(t, "dev-terraform-state", data.S3BucketName)
	})

	t.Run("render bucket name template with multiple variables", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{
						"prd": "987654321098",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "{{.AppDir}}-{{.Env}}-{{.Region}}-tfstate",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		data, err := gen.prepareTemplateData("prd", "eu-central-1", "webapp")
		assert.NoError(t, err)
		assert.Equal(t, "webapp-prd-eu-central-1-tfstate", data.S3BucketName)
	})

	t.Run("render bucket name without template syntax", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "static-bucket-name",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		data, err := gen.prepareTemplateData("dev", "us-east-1", "myapp")
		assert.NoError(t, err)
		assert.Equal(t, "static-bucket-name", data.S3BucketName)
	})

	t.Run("error on invalid template syntax", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "{{.Env",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		_, err = gen.prepareTemplateData("dev", "us-east-1", "myapp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to render bucket_name template")
	})

	t.Run("bucket name with template functions", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					AccountMapping: map[string]string{
						"dev": "123456789012",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "{{.Env | toUpper}}-terraform-state",
				},
			},
		}
		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		data, err := gen.prepareTemplateData("dev", "us-east-1", "myapp")
		assert.NoError(t, err)
		assert.Equal(t, "DEV-terraform-state", data.S3BucketName)
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
		assert.Contains(t, changes[0], "added tag - team: platform")
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
		assert.Contains(t, changes[0], "initialized configuration tracking")
	})

	t.Run("returns true when file has malformed tags metadata and config has no tags", func(t *testing.T) {
		t.Helper()

		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		// File with malformed tags JSON (trailing comma)
		versionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {"managed_by": "terraform", }
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
			DefaultTags:        make(map[string]string), // No tags in config
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.True(t, needsUpdate)
		assert.Contains(t, changes, "repaired metadata",
			"Should detect and fix malformed tags metadata")
	})

	t.Run("returns true when file has malformed tags metadata and config has tags", func(t *testing.T) {
		t.Helper()

		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		// File with invalid tag JSON
		versionsContent := `## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {invalid json}
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
			DefaultTags: map[string]string{
				"managed_by": "terraform",
				"team":       "platform",
			},
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.True(t, needsUpdate)
		// When config has tags and file has malformed tags, should show tags being added
		assert.Contains(t, changes[0], "added tag",
			"Should show specific tags being added when fixing malformed metadata")
	})

	t.Run("returns false when file has no tags metadata and config has no tags", func(t *testing.T) {
		t.Helper()

		cfg := &config.Config{}
		filesystem := fs.NewMemoryFileSystem()
		gen := NewGenerator(cfg, filesystem, logger.New(false))

		// File without tags metadata at all
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
			DefaultTags:        make(map[string]string), // No tags in config
		}

		needsUpdate, changes, err := gen.shouldUpdateVersions("versions.tf", data)
		assert.NoError(t, err)
		assert.False(t, needsUpdate,
			"Should not update when both file and config have no tags")
		assert.Empty(t, changes)
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
## tfskel-tags: {"managed_by": "terraform"}

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
			name: "extract tags metadata with mixed-case keys - normalizes to lowercase",
			content: `## This file is auto generated by tfskel
## tfskel-metadata: {"tf_ver": "~> 1.13", "aws_provider_ver": "~> 6.0"}
## tfskel-tags: {"ManagedBy": "terraform", "Team": "platform", "Environment": "production"}

terraform {
  required_version = "~> 1.13"
}`,
			metadataKey: "tags",
			expected:    map[string]string{"managedby": "terraform", "team": "platform", "environment": "production"},
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

func TestGenerator_prepareTemplateData(t *testing.T) {
	tests := []struct {
		name                 string
		config               *config.Config
		env                  string
		region               string
		appDir               string
		expectedTags         map[string]string
		expectedTagsJSON     string
		expectedProviderVer  string
		expectedTerraformVer string
		expectedS3Bucket     string
		expectError          bool
	}{
		{
			name: "normalizes mixed-case tag keys to lowercase",
			config: &config.Config{
				TerraformVersion: "~> 1.13",
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						Version: "~> 6.0",
						DefaultTags: map[string]string{
							"ManagedBy":   "terraform",
							"Team":        "platform",
							"Environment": "production",
							"Owner":       "devops",
						},
						AccountMapping: map[string]string{"dev": "123456789012"},
					},
				},
				Backend: &config.Backend{
					S3: &config.S3Backend{BucketName: "test-bucket"},
				},
			},
			env:                  "dev",
			region:               "eu-central-1",
			appDir:               "test-app",
			expectedTags:         map[string]string{"managedby": "terraform", "team": "platform", "environment": "production", "owner": "devops"},
			expectedTagsJSON:     `{"environment":"production","managedby":"terraform","owner":"devops","team":"platform"}`,
			expectedProviderVer:  "~> 6.0",
			expectedTerraformVer: "~> 1.13",
			expectedS3Bucket:     "test-bucket",
			expectError:          false,
		},
		{
			name: "empty tags produce empty JSON object",
			config: &config.Config{
				TerraformVersion: "~> 1.13",
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						Version:        "~> 6.0",
						DefaultTags:    nil,
						AccountMapping: map[string]string{"dev": "123456789012"},
					},
				},
				Backend: &config.Backend{
					S3: &config.S3Backend{BucketName: "test-bucket"},
				},
			},
			env:                  "dev",
			region:               "eu-central-1",
			appDir:               "test-app",
			expectedTags:         map[string]string{},
			expectedTagsJSON:     "{}",
			expectedProviderVer:  "~> 6.0",
			expectedTerraformVer: "~> 1.13",
			expectedS3Bucket:     "test-bucket",
			expectError:          false,
		},
		{
			name: "tags with only lowercase keys are preserved",
			config: &config.Config{
				TerraformVersion: "~> 1.13",
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						Version: "~> 6.0",
						DefaultTags: map[string]string{
							"managed_by": "terraform",
							"team":       "platform",
						},
						AccountMapping: map[string]string{"dev": "123456789012"},
					},
				},
				Backend: &config.Backend{
					S3: &config.S3Backend{BucketName: "test-bucket"},
				},
			},
			env:                  "dev",
			region:               "eu-central-1",
			appDir:               "test-app",
			expectedTags:         map[string]string{"managed_by": "terraform", "team": "platform"},
			expectedTagsJSON:     `{"managed_by":"terraform","team":"platform"}`,
			expectedProviderVer:  "~> 6.0",
			expectedTerraformVer: "~> 1.13",
			expectedS3Bucket:     "test-bucket",
			expectError:          false,
		},
		{
			name: "uses default values when provider config is nil",
			config: &config.Config{
				TerraformVersion: "~> 1.13",
				Provider:         nil,
				Backend: &config.Backend{
					S3: &config.S3Backend{BucketName: "test-bucket"},
				},
			},
			env:                  "dev",
			region:               "us-east-1",
			appDir:               "my-app",
			expectedTags:         map[string]string{},
			expectedTagsJSON:     "{}",
			expectedProviderVer:  "~> 6.0", // default value
			expectedTerraformVer: "~> 1.13",
			expectedS3Bucket:     "test-bucket",
			expectError:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Helper()

			filesystem := fs.NewMemoryFileSystem()
			log := logger.New(false)

			gen := NewGenerator(tt.config, filesystem, log)

			data, err := gen.prepareTemplateData(tt.env, tt.region, tt.appDir)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, data)

			// Verify tag normalization
			assert.Equal(t, tt.expectedTags, data.DefaultTags,
				"Tags should be normalized to lowercase")

			// Verify DefaultTagsJSON field
			assert.Equal(t, tt.expectedTagsJSON, data.DefaultTagsJSON,
				"DefaultTagsJSON should match expected JSON output")

			// Verify other fields are set correctly
			assert.Equal(t, tt.expectedProviderVer, data.AWSProviderVersion)
			assert.Equal(t, tt.expectedTerraformVer, data.TerraformVersion)
			assert.Equal(t, tt.expectedS3Bucket, data.S3BucketName)
			assert.Equal(t, tt.env, data.Env)
			assert.Equal(t, tt.region, data.Region)
			assert.Equal(t, tt.appDir, data.AppDir)
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
			expectedMsgs:  []string{"added tag - team: platform"},
		},
		{
			name:          "tag removed",
			fileTags:      map[string]string{"managed_by": "terraform", "team": "platform"},
			configTags:    map[string]string{"managed_by": "terraform"},
			expectChanges: true,
			expectedMsgs:  []string{"removed tag - team (was: platform)"},
		},
		{
			name:          "tag value changed",
			fileTags:      map[string]string{"managed_by": "terraform", "team": "platform"},
			configTags:    map[string]string{"managed_by": "terraform", "team": "devops"},
			expectChanges: true,
			expectedMsgs:  []string{"changed tag - team: platform -> devops"},
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

func TestGenerator_generateWorkflowFileName(t *testing.T) {
	tests := []struct {
		name             string
		originalFileName string
		data             *templates.Data
		config           *config.Config
		expectedOutput   string
		expectError      bool
	}{
		{
			name:             "lint workflow with standard data",
			originalFileName: "lint.yaml",
			data: &templates.Data{
				AppDir:      "myapp",
				Env:         "dev",
				ShortRegion: "euc1",
			},
			config:         &config.Config{},
			expectedOutput: "myapp-dev-euc1-lint.yaml",
			expectError:    false,
		},
		{
			name:             "terraform workflow with standard data",
			originalFileName: "terraform.yaml",
			data: &templates.Data{
				AppDir:      "backend-api",
				Env:         "prd",
				ShortRegion: "use1",
			},
			config:         &config.Config{},
			expectedOutput: "backend-api-prd-use1-terraform.yaml",
			expectError:    false,
		},
		{
			name:             "workflow with hyphenated app name",
			originalFileName: "lint.yaml",
			data: &templates.Data{
				AppDir:      "my-complex-app",
				Env:         "stg",
				ShortRegion: "euw1",
			},
			config:         &config.Config{},
			expectedOutput: "my-complex-app-stg-euw1-lint.yaml",
			expectError:    false,
		},
		{
			name:             "workflow with underscore app name",
			originalFileName: "terraform.yaml",
			data: &templates.Data{
				AppDir:      "test_app",
				Env:         "dev",
				ShortRegion: "apse2",
			},
			config:         &config.Config{},
			expectedOutput: "test_app-dev-apse2-terraform.yaml",
			expectError:    false,
		},
		{
			name:             "custom name template renders correctly",
			originalFileName: "lint.yaml",
			data: &templates.Data{
				AppDir:      "myapp",
				Env:         "dev",
				Region:      "us-east-1",
				ShortRegion: "use1",
			},
			config: &config.Config{
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						NameTemplate: "{{.AppDir}}-{{.Env}}",
					},
				},
			},
			expectedOutput: "myapp-dev-lint.yaml",
			expectError:    false,
		},
		{
			name:             "invalid custom template returns error",
			originalFileName: "lint.yaml",
			data: &templates.Data{
				AppDir:      "myapp",
				Env:         "dev",
				ShortRegion: "use1",
			},
			config: &config.Config{
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						NameTemplate: "{{.AppDir",
					},
				},
			},
			expectedOutput: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filesystem := fs.NewMemoryFileSystem()
			log := logger.New(false)
			gen := NewGenerator(tt.config, filesystem, log)

			result, err := gen.generateWorkflowFileName(tt.originalFileName, tt.data)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, result)
			}
		})
	}
}

func TestGenerator_determineOutputPath_GitHubWorkflows(t *testing.T) {
	tests := []struct {
		name         string
		tmplPath     string
		appPath      string
		data         *templates.Data
		expectedPath string
		expectedOK   bool
	}{
		{
			name:     "github lint workflow creates dynamic name",
			tmplPath: "github/lint.yaml.tmpl",
			appPath:  "envs/dev/eu-central-1/myapp",
			data: &templates.Data{
				AppDir:      "myapp",
				Env:         "dev",
				ShortRegion: "euc1",
			},
			expectedPath: ".github/workflows/myapp-dev-euc1-lint.yaml",
			expectedOK:   true,
		},
		{
			name:     "github terraform workflow creates dynamic name",
			tmplPath: "github/terraform.yaml.tmpl",
			appPath:  "envs/prd/us-west-2/api",
			data: &templates.Data{
				AppDir:      "api",
				Env:         "prd",
				ShortRegion: "usw2",
			},
			expectedPath: ".github/workflows/api-prd-usw2-terraform.yaml",
			expectedOK:   true,
		},
		{
			name:     "tf template goes to app directory",
			tmplPath: "tf/backend.tf.tmpl",
			appPath:  "envs/dev/eu-central-1/myapp",
			data: &templates.Data{
				AppDir:      "myapp",
				Env:         "dev",
				ShortRegion: "euc1",
			},
			expectedPath: "envs/dev/eu-central-1/myapp/backend.tf",
			expectedOK:   true,
		},
		{
			name:     "root template goes to project root",
			tmplPath: "root/.gitignore.tmpl",
			appPath:  "envs/dev/eu-central-1/myapp",
			data: &templates.Data{
				AppDir:      "myapp",
				Env:         "dev",
				ShortRegion: "euc1",
			},
			expectedPath: ".gitignore",
			expectedOK:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{}
			filesystem := fs.NewMemoryFileSystem()
			log := logger.New(false)
			gen := NewGenerator(cfg, filesystem, log)

			resultPath, ok := gen.determineOutputPath(tt.tmplPath, tt.appPath, tt.data)
			assert.Equal(t, tt.expectedOK, ok)
			if ok {
				assert.Equal(t, tt.expectedPath, resultPath)
			}
		})
	}
}

func TestGenerator_GitHubWorkflows_Integration(t *testing.T) {
	t.Run("creates github workflows when flag is enabled", func(t *testing.T) {
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
					BucketName: "test-bucket",
				},
			},
			Generate: &config.Generate{
				GithubWorkflows: &config.GithubWorkflows{
					Create: true,
				},
			},
		}

		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/eu-central-1/testapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "eu-central-1", "testapp")
		require.NoError(t, err)

		// Verify terraform files were created
		assert.True(t, filesystem.FileExists(filepath.Join(appPath, "backend.tf")))
		assert.True(t, filesystem.FileExists(filepath.Join(appPath, "versions.tf")))

		// Verify github workflow files were created with correct dynamic names
		expectedLintWorkflow := ".github/workflows/testapp-dev-euc1-lint.yaml"
		expectedTerraformWorkflow := ".github/workflows/testapp-dev-euc1-terraform.yaml"

		assert.True(t, filesystem.FileExists(expectedLintWorkflow),
			"expected lint workflow to exist at %s", expectedLintWorkflow)
		assert.True(t, filesystem.FileExists(expectedTerraformWorkflow),
			"expected terraform workflow to exist at %s", expectedTerraformWorkflow)

		// Verify workflow files have content
		lintContent, err := filesystem.ReadFile(expectedLintWorkflow)
		assert.NoError(t, err)
		assert.NotEmpty(t, lintContent, "lint workflow should have content")

		terraformContent, err := filesystem.ReadFile(expectedTerraformWorkflow)
		assert.NoError(t, err)
		assert.NotEmpty(t, terraformContent, "terraform workflow should have content")
	})

	t.Run("skips github workflows when flag is disabled", func(t *testing.T) {
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
					BucketName: "test-bucket",
				},
			},
			Generate: &config.Generate{
				GithubWorkflows: &config.GithubWorkflows{
					Create: false,
				},
			},
		}

		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/eu-central-1/testapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "eu-central-1", "testapp")
		require.NoError(t, err)

		// Verify terraform files were created
		assert.True(t, filesystem.FileExists(filepath.Join(appPath, "backend.tf")))
		assert.True(t, filesystem.FileExists(filepath.Join(appPath, "versions.tf")))

		// Verify github workflow files were NOT created
		expectedLintWorkflow := ".github/workflows/testapp-dev-euc1-lint.yaml"
		expectedTerraformWorkflow := ".github/workflows/testapp-dev-euc1-terraform.yaml"

		assert.False(t, filesystem.FileExists(expectedLintWorkflow),
			"lint workflow should not exist when flag is disabled")
		assert.False(t, filesystem.FileExists(expectedTerraformWorkflow),
			"terraform workflow should not exist when flag is disabled")

		// Verify .github/workflows directory was not created
		assert.False(t, filesystem.DirExists(".github/workflows"),
			".github/workflows directory should not be created when flag is disabled")
	})

	t.Run("skips github workflows when Generate config is nil", func(t *testing.T) {
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
					BucketName: "test-bucket",
				},
			},
			Generate: nil, // No Generate config provided
		}

		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/eu-central-1/testapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "eu-central-1", "testapp")
		require.NoError(t, err)

		// Verify terraform files were created
		assert.True(t, filesystem.FileExists(filepath.Join(appPath, "backend.tf")))
		assert.True(t, filesystem.FileExists(filepath.Join(appPath, "versions.tf")))

		// Verify github workflow files were NOT created
		expectedLintWorkflow := ".github/workflows/testapp-dev-euc1-lint.yaml"
		expectedTerraformWorkflow := ".github/workflows/testapp-dev-euc1-terraform.yaml"

		assert.False(t, filesystem.FileExists(expectedLintWorkflow),
			"lint workflow should not exist when Generate config is nil")
		assert.False(t, filesystem.FileExists(expectedTerraformWorkflow),
			"terraform workflow should not exist when Generate config is nil")
	})

	t.Run("does not overwrite existing workflow files", func(t *testing.T) {
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
					BucketName: "test-bucket",
				},
			},
			Generate: &config.Generate{
				GithubWorkflows: &config.GithubWorkflows{
					Create: true,
				},
			},
		}

		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		appPath := "envs/dev/eu-central-1/testapp"
		_ = filesystem.MkdirAll(appPath, 0755)

		// Create existing workflow file with custom content
		existingWorkflowPath := ".github/workflows/testapp-dev-euc1-lint.yaml"
		existingContent := "# Custom workflow content - do not overwrite"
		_ = filesystem.MkdirAll(filepath.Dir(existingWorkflowPath), 0755)
		err := filesystem.WriteFile(existingWorkflowPath, []byte(existingContent), 0644)
		require.NoError(t, err)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		err = gen.generateFiles(appPath, "dev", "eu-central-1", "testapp")
		require.NoError(t, err)

		// Verify existing workflow was NOT overwritten
		content, err := filesystem.ReadFile(existingWorkflowPath)
		require.NoError(t, err)
		assert.Equal(t, existingContent, string(content),
			"existing workflow file should not be overwritten")
	})

	t.Run("creates workflows with different names for different environments", func(t *testing.T) {
		cfg := &config.Config{
			TerraformVersion: "~> 1.13",
			Provider: &config.Provider{
				AWS: &config.AWSProvider{
					Version: "~> 6.0",
					AccountMapping: map[string]string{
						"dev": "123456789012",
						"prd": "234567890123",
					},
				},
			},
			Backend: &config.Backend{
				S3: &config.S3Backend{
					BucketName: "test-bucket",
				},
			},
			Generate: &config.Generate{
				GithubWorkflows: &config.GithubWorkflows{
					Create: true,
				},
			},
		}

		filesystem := fs.NewMemoryFileSystem()
		log := logger.New(false)
		gen := NewGenerator(cfg, filesystem, log)

		renderer, err := templates.NewRenderer()
		require.NoError(t, err)
		gen.renderer = renderer

		// Generate for dev environment
		devAppPath := "envs/dev/eu-central-1/myapp"
		_ = filesystem.MkdirAll(devAppPath, 0755)
		err = gen.generateFiles(devAppPath, "dev", "eu-central-1", "myapp")
		require.NoError(t, err)

		// Generate for prd environment
		prdAppPath := "envs/prd/us-east-1/myapp"
		_ = filesystem.MkdirAll(prdAppPath, 0755)
		err = gen.generateFiles(prdAppPath, "prd", "us-east-1", "myapp")
		require.NoError(t, err)

		// Verify both sets of workflows were created with different names
		devLintWorkflow := ".github/workflows/myapp-dev-euc1-lint.yaml"
		devTerraformWorkflow := ".github/workflows/myapp-dev-euc1-terraform.yaml"
		prdLintWorkflow := ".github/workflows/myapp-prd-use1-lint.yaml"
		prdTerraformWorkflow := ".github/workflows/myapp-prd-use1-terraform.yaml"

		assert.True(t, filesystem.FileExists(devLintWorkflow),
			"dev lint workflow should exist")
		assert.True(t, filesystem.FileExists(devTerraformWorkflow),
			"dev terraform workflow should exist")
		assert.True(t, filesystem.FileExists(prdLintWorkflow),
			"prd lint workflow should exist")
		assert.True(t, filesystem.FileExists(prdTerraformWorkflow),
			"prd terraform workflow should exist")

		// Verify they are different files
		devContent, _ := filesystem.ReadFile(devLintWorkflow)
		prdContent, _ := filesystem.ReadFile(prdLintWorkflow)
		assert.NotEmpty(t, devContent)
		assert.NotEmpty(t, prdContent)
	})
}

func TestGenerator_buildAWSRoleArn(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		data        *templates.Data
		expectedArn string
		expectError bool
	}{
		{
			name: "explicit ARN takes precedence over role name",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleArn:  "arn:aws:iam::999999999999:role/CustomRole",
						AWSRoleName: "ShouldBeIgnored",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::999999999999:role/CustomRole",
			expectError: false,
		},
		{
			name: "role name constructs ARN when no explicit ARN provided",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "TerraformDeployRole",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/TerraformDeployRole",
			expectError: false,
		},
		{
			name: "returns placeholder when no ARN or role name provided",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/REPLACE_WITH_ROLE_TO_ASSUME",
			expectError: false,
		},
		{
			name: "returns placeholder when GithubWorkflows is nil",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: nil,
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/REPLACE_WITH_ROLE_TO_ASSUME",
			expectError: false,
		},
		{
			name: "returns placeholder when Generate is nil",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: nil,
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/REPLACE_WITH_ROLE_TO_ASSUME",
			expectError: false,
		},
		{
			name: "works with different environment",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
							"prd": "987654321098",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "TerraformDeployRole",
					},
				},
			},
			data: &templates.Data{
				Env:         "prd",
				Region:      "eu-west-1",
				AppDir:      "webapp",
				AccountID:   "987654321098",
				ShortRegion: "euw1",
			},
			expectedArn: "arn:aws:iam::987654321098:role/TerraformDeployRole",
			expectError: false,
		},
		{
			name: "ARN with template syntax renders correctly",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleArn: "arn:aws:iam::{{.AccountID}}:role/terraform-{{.Env | toUpper}}-role",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/terraform-DEV-role",
			expectError: false,
		},
		{
			name: "role name with template syntax renders correctly",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "terraform-{{.Env}}-deploy-role",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/terraform-dev-deploy-role",
			expectError: false,
		},
		{
			name: "role name with uppercase template function",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"prd": "987654321098",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "TERRAFORM-{{.Env | toUpper}}-ROLE",
					},
				},
			},
			data: &templates.Data{
				Env:         "prd",
				Region:      "eu-west-1",
				AppDir:      "webapp",
				AccountID:   "987654321098",
				ShortRegion: "euw1",
			},
			expectedArn: "arn:aws:iam::987654321098:role/TERRAFORM-PRD-ROLE",
			expectError: false,
		},
		{
			name: "role name with region template variable",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "terraform-{{.Env}}-{{.ShortRegion}}-role",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "eu-central-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "euc1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/terraform-dev-euc1-role",
			expectError: false,
		},
		{
			name: "role name with AppDir template variable",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "{{.AppDir}}-{{.Env}}-deploy",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "webapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::123456789012:role/webapp-dev-deploy",
			expectError: false,
		},
		{
			name: "invalid ARN template returns error",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleArn: "arn:aws:iam::{{.AccountID:role/invalid",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "",
			expectError: true,
		},
		{
			name: "invalid role name template returns error",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleName: "terraform-{{.Env",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "",
			expectError: true,
		},
		{
			name: "plain ARN without template syntax",
			config: &config.Config{
				Provider: &config.Provider{
					AWS: &config.AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Generate: &config.Generate{
					GithubWorkflows: &config.GithubWorkflows{
						AWSRoleArn: "arn:aws:iam::999999999999:role/StaticRole",
					},
				},
			},
			data: &templates.Data{
				Env:         "dev",
				Region:      "us-east-1",
				AppDir:      "myapp",
				AccountID:   "123456789012",
				ShortRegion: "use1",
			},
			expectedArn: "arn:aws:iam::999999999999:role/StaticRole",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filesystem := fs.NewMemoryFileSystem()
			log := logger.New(false)
			gen := NewGenerator(tt.config, filesystem, log)

			// Initialize renderer for template rendering
			renderer, err := templates.NewRenderer()
			require.NoError(t, err)
			gen.renderer = renderer

			result, err := gen.buildAWSRoleArn(tt.data)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid template syntax")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedArn, result)
			}
		})
	}
}

func TestSanitizeWorkflowFileName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedValid bool
		expectedOut   string
	}{
		{
			name:          "valid simple filename",
			input:         "myapp-dev-euc1-lint.yaml",
			expectedValid: true,
			expectedOut:   "myapp-dev-euc1-lint.yaml",
		},
		{
			name:          "valid filename with underscores",
			input:         "my_app-dev-lint.yaml",
			expectedValid: true,
			expectedOut:   "my_app-dev-lint.yaml",
		},
		{
			name:          "reject path with directory traversal using ..",
			input:         "../../../etc/passwd",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "reject path with subdirectory",
			input:         "subdir/workflow.yaml",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "reject absolute path",
			input:         "/etc/passwd",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "reject path with .. in middle",
			input:         "valid-name..yaml",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "reject empty filename",
			input:         "",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "reject just extension",
			input:         ".yaml",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "reject path even if basename would be valid",
			input:         "some/path/workflow.yaml",
			expectedValid: false,
			expectedOut:   "",
		},
		{
			name:          "accept filename with dots",
			input:         "my.app.workflow.yaml",
			expectedValid: true,
			expectedOut:   "my.app.workflow.yaml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, valid := sanitizeWorkflowFileName(tt.input)
			assert.Equal(t, tt.expectedValid, valid, "validity mismatch")
			if valid {
				assert.Equal(t, tt.expectedOut, result, "sanitized output mismatch")
			}
		})
	}
}
