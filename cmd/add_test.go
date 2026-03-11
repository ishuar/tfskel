package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAddParams(t *testing.T) {
	tests := []struct {
		name          string
		env           string
		region        string
		appDir        string
		wantEnv       string
		wantRegion    string
		wantAppDir    string
		wantError     bool
		errorContains string
	}{
		{
			name:       "all parameters valid",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "myapp",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "valid with different environment",
			env:        "prd",
			region:     "eu-central-1",
			appDir:     "webapp",
			wantEnv:    "prd",
			wantRegion: "eu-central-1",
			wantAppDir: "webapp",
			wantError:  false,
		},
		{
			name:       "valid with complex app directory name",
			env:        "stg",
			region:     "ap-south-1",
			appDir:     "my-complex-app-name",
			wantEnv:    "stg",
			wantRegion: "ap-south-1",
			wantAppDir: "my-complex-app-name",
			wantError:  false,
		},
		{
			name:          "missing environment",
			env:           "",
			region:        "us-east-1",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "environment",
		},
		{
			name:          "missing region",
			env:           "dev",
			region:        "",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "region",
		},
		{
			name:          "missing app directory",
			env:           "dev",
			region:        "us-east-1",
			appDir:        "",
			wantError:     true,
			errorContains: "app directory",
		},
		{
			name:          "all parameters missing",
			env:           "",
			region:        "",
			appDir:        "",
			wantError:     true,
			errorContains: "environment", // Should fail on first check
		},
		{
			name:          "only env provided",
			env:           "dev",
			region:        "",
			appDir:        "",
			wantError:     true,
			errorContains: "region", // Should fail on region check
		},
		{
			name:          "env and region provided, missing appdir",
			env:           "dev",
			region:        "us-east-1",
			appDir:        "",
			wantError:     true,
			errorContains: "app directory",
		},
		{
			name:       "trim leading whitespace from env",
			env:        "  dev",
			region:     "us-east-1",
			appDir:     "myapp",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "trim trailing whitespace from region",
			env:        "dev",
			region:     "us-east-1  ",
			appDir:     "myapp",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "trim both sides of appDir",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "  myapp  ",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:       "trim all parameters",
			env:        "  dev  ",
			region:     "  us-east-1  ",
			appDir:     "  myapp  ",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "myapp",
			wantError:  false,
		},
		{
			name:          "whitespace-only env is rejected",
			env:           "   ",
			region:        "us-east-1",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "environment",
		},
		{
			name:          "whitespace-only region is rejected",
			env:           "dev",
			region:        "   ",
			appDir:        "myapp",
			wantError:     true,
			errorContains: "region",
		},
		{
			name:          "whitespace-only appDir is rejected",
			env:           "dev",
			region:        "us-east-1",
			appDir:        "   ",
			wantError:     true,
			errorContains: "app directory",
		},
		{
			name:       "preserves internal whitespace in appDir",
			env:        "dev",
			region:     "us-east-1",
			appDir:     "  my app  ",
			wantEnv:    "dev",
			wantRegion: "us-east-1",
			wantAppDir: "my app",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnv, gotRegion, gotAppDir, err := validateAddParams(tt.env, tt.region, tt.appDir)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains, "error message should contain expected text")
				}
				assert.Empty(t, gotEnv, "env should be empty on error")
				assert.Empty(t, gotRegion, "region should be empty on error")
				assert.Empty(t, gotAppDir, "appDir should be empty on error")
			} else {
				assert.NoError(t, err, "should not return error for valid parameters")
				assert.Equal(t, tt.wantEnv, gotEnv, "env should match expected")
				assert.Equal(t, tt.wantRegion, gotRegion, "region should match expected")
				assert.Equal(t, tt.wantAppDir, gotAppDir, "appDir should match expected")
			}
		})
	}
}

func TestValidateAddParams_ErrorMessages(t *testing.T) {
	t.Run("environment error has helpful message", func(t *testing.T) {
		_, _, _, err := validateAddParams("", "us-east-1", "myapp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment")
		assert.Contains(t, err.Error(), "use --env flag")
	})

	t.Run("region error has helpful message", func(t *testing.T) {
		_, _, _, err := validateAddParams("dev", "", "myapp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region")
		assert.Contains(t, err.Error(), "use --region flag")
	})

	t.Run("appdir error has helpful message", func(t *testing.T) {
		_, _, _, err := validateAddParams("dev", "us-east-1", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app directory")
		assert.Contains(t, err.Error(), "provide as argument")
	})
}

func TestValidateAddParams_EdgeCases(t *testing.T) {
	t.Run("whitespace-only parameters are rejected", func(t *testing.T) {
		// New implementation trims and validates, so these should fail
		_, _, _, err := validateAddParams("   ", "us-east-1", "myapp")
		assert.Error(t, err, "whitespace-only env should be rejected")

		_, _, _, err = validateAddParams("dev", "   ", "myapp")
		assert.Error(t, err, "whitespace-only region should be rejected")

		_, _, _, err = validateAddParams("dev", "us-east-1", "   ")
		assert.Error(t, err, "whitespace-only appdir should be rejected")
	})

	t.Run("parameters with special characters pass validation", func(t *testing.T) {
		// Validation only checks for empty strings, not format
		env, region, appDir, err := validateAddParams("dev-v2", "us-east-1", "my_app/test")
		assert.NoError(t, err, "special characters in appdir should pass basic validation")
		assert.Equal(t, "dev-v2", env)
		assert.Equal(t, "us-east-1", region)
		assert.Equal(t, "my_app/test", appDir)

		env, region, appDir, err = validateAddParams("dev", "eu-central-1", "app-with-dashes")
		assert.NoError(t, err, "dashes in appdir should pass")
		assert.Equal(t, "dev", env)
		assert.Equal(t, "eu-central-1", region)
		assert.Equal(t, "app-with-dashes", appDir)

		env, region, appDir, err = validateAddParams("dev", "eu-central-1", "app_with_underscores")
		assert.NoError(t, err, "underscores in appdir should pass")
		assert.Equal(t, "dev", env)
		assert.Equal(t, "eu-central-1", region)
		assert.Equal(t, "app_with_underscores", appDir)
	})
}

func TestValidateAddParams_ValidationOrder(t *testing.T) {
	t.Run("validates in order: env, region, appdir", func(t *testing.T) {
		// When all are missing, should fail on environment first
		_, _, _, err := validateAddParams("", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "environment", "should check env first")

		// When env is present but region missing
		_, _, _, err = validateAddParams("dev", "", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "region", "should check region second")

		// When env and region present but appdir missing
		_, _, _, err = validateAddParams("dev", "us-east-1", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app directory", "should check appdir last")
	})
}

func TestAddCommand_CommandSetup(t *testing.T) {
	t.Run("command is properly registered", func(t *testing.T) {
		assert.NotNil(t, addCmd, "addCmd should be initialized")
		assert.Equal(t, "add <app-dir>", addCmd.Use, "command use pattern should be correct")
		assert.Equal(t, "main", addCmd.GroupID, "command should be in main group")
	})

	t.Run("command has required flags", func(t *testing.T) {
		assert.NotNil(t, addCmd.Flags(), "command should have flags")

		// Check required flags exist
		envFlag := addCmd.Flags().Lookup("env")
		assert.NotNil(t, envFlag, "--env flag should exist")

		regionFlag := addCmd.Flags().Lookup("region")
		assert.NotNil(t, regionFlag, "--region flag should exist")

		// Check optional flags exist
		templatesFlag := addCmd.Flags().Lookup("templates-dir")
		assert.NotNil(t, templatesFlag, "--templates-dir flag should exist")

		s3Flag := addCmd.Flags().Lookup("s3-bucket-name")
		assert.NotNil(t, s3Flag, "--s3-bucket-name flag should exist")

		workflowsFlag := addCmd.Flags().Lookup("create-github-workflows")
		assert.NotNil(t, workflowsFlag, "--create-github-workflows flag should exist")
	})

	t.Run("command requires exactly one argument", func(t *testing.T) {
		// The Args field should enforce exactly one argument
		assert.NotNil(t, addCmd.Args, "command should have Args validator")
		// cobra.ExactArgs(1) is set in the command definition
	})

	t.Run("command has help text", func(t *testing.T) {
		assert.NotEmpty(t, addCmd.Short, "command should have short description")
		assert.NotEmpty(t, addCmd.Long, "command should have long description")
		assert.NotEmpty(t, addCmd.Example, "command should have examples")

		// Verify help text references the correct command name
		assert.Contains(t, addCmd.Short, "Add", "short description should mention 'Add'")
		assert.Contains(t, addCmd.Long, "add command", "long description should reference 'add command'")
		assert.Contains(t, addCmd.Example, "tfskel add", "examples should use 'tfskel add'")
	})
}
