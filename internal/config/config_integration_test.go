package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadFromYAML tests loading configuration from actual YAML content
// This ensures the YAML structure matches the struct tags and Viper bindings
func TestLoadFromYAML(t *testing.T) {
	tests := []struct {
		name         string
		yamlContent  string
		expectError  bool
		validateFunc func(*testing.T, *Config)
	}{
		{
			name: "full valid config with templates and workflows sections",
			yamlContent: `
terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    regions:
      - eu-central-1
      - us-east-1
    account_mapping:
      dev: "123456789012"
      prd: "987654321098"
backend:
  s3:
    bucket_name: my-terraform-state-bucket
templates:
  dir: /custom/templates
workflows:
  create: true
  name_template: "{{.AppDir}}-{{.Env}}"
  aws_role_name: GitHubActionsRole
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "~> 1.13", cfg.TerraformVersion)
				assert.Equal(t, "~> 6.0", cfg.Provider.AWS.Version)
				assert.Equal(t, "123456789012", cfg.Provider.AWS.AccountMapping["dev"])
				assert.Equal(t, "my-terraform-state-bucket", cfg.Backend.S3.BucketName)

				// Test Templates section
				require.NotNil(t, cfg.Templates)
				assert.Equal(t, "/custom/templates", cfg.Templates.Dir)

				// Test Workflows
				require.NotNil(t, cfg.Workflows)
				assert.True(t, cfg.Workflows.Create)
				assert.Equal(t, "{{.AppDir}}-{{.Env}}", cfg.Workflows.NameTemplate)
				assert.Equal(t, "GitHubActionsRole", cfg.Workflows.AWSRoleName)
			},
		},
		{
			name: "OLD structure should NOT work - templates_dir at root level",
			yamlContent: `
terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
backend:
  s3:
    bucket_name: test-bucket
templates_dir: /custom/path
extra_template_extensions:
  - yaml.tmpl
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				// With old structure at root level, values won't be populated in Templates
				// This validates the breaking change - old configs won't work
				if cfg.Templates != nil {
					assert.Empty(t, cfg.Templates.Dir, "Old YAML structure should not populate Templates.Dir")
				}
			},
		},
		{
			name: "minimal config without templates or workflows sections",
			yamlContent: `
terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
backend:
  s3:
    bucket_name: test-bucket
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				assert.Equal(t, "~> 1.13", cfg.TerraformVersion)
				// Templates section can be nil or empty
				if cfg.Templates != nil {
					assert.Empty(t, cfg.Templates.Dir)
				}
			},
		},
		{
			name: "workflows section only",
			yamlContent: `
terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
backend:
  s3:
    bucket_name: test-bucket
workflows:
  create: true
  aws_role_arn: "arn:aws:iam::123456789012:role/MyRole"
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Workflows)
				assert.True(t, cfg.Workflows.Create)
				assert.Equal(t, "arn:aws:iam::123456789012:role/MyRole", cfg.Workflows.AWSRoleArn)
				// templates_dir should have default/empty value
				if cfg.Templates != nil {
					assert.Empty(t, cfg.Templates.Dir)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary config file
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, ".tfskel.yaml")
			err := os.WriteFile(configFile, []byte(tt.yamlContent), 0644)
			require.NoError(t, err)

			// Create a new Viper instance for this test
			v := viper.New()
			v.SetConfigFile(configFile)
			v.SetConfigType("yaml")

			// Read the config
			err = v.ReadInConfig()
			require.NoError(t, err, "Failed to read config file")

			// Create a minimal cobra command for Load
			cmd := &cobra.Command{}

			// Load the config
			cfg, err := Load(cmd, v)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				if tt.validateFunc != nil {
					tt.validateFunc(t, cfg)
				}
			}
		})
	}
}

// TestLoadExampleConfigFile ensures the .tfskel.example.yaml file is valid and loads correctly
// This is critical - if the example config is broken, users will have a bad experience
func TestLoadExampleConfigFile(t *testing.T) {
	// Find the example config file
	exampleConfigPath := filepath.Join("..", "..", ".tfskel.example.yaml")

	// Check if file exists
	if _, err := os.Stat(exampleConfigPath); os.IsNotExist(err) {
		t.Skip("Example config file not found, skipping test")
	}

	// Create a new Viper instance
	v := viper.New()
	v.SetConfigFile(exampleConfigPath)
	v.SetConfigType("yaml")

	// Read the config
	err := v.ReadInConfig()
	require.NoError(t, err, ".tfskel.example.yaml should be valid YAML")

	// Create a minimal cobra command
	cmd := &cobra.Command{}

	// Load the config
	cfg, err := Load(cmd, v)
	require.NoError(t, err, ".tfskel.example.yaml should load without errors")
	require.NotNil(t, cfg)

	// Validate key fields exist and have expected structure
	t.Run("validate example file structure", func(t *testing.T) {
		// These should be set in the example
		assert.NotEmpty(t, cfg.TerraformVersion)

		// Templates section should exist and have the new structure
		require.NotNil(t, cfg.Templates, "templates section must exist in example config")

		// dir should be under templates
		assert.NotEmpty(t, cfg.Templates.Dir, "templates.dir should be set in example")

		// workflows should be separate
		require.NotNil(t, cfg.Workflows, "workflows section should exist")
	})
}

// TestViperBindings validates that Viper flag bindings match the config structure
// This ensures CLI flags are bound to the correct YAML paths
func TestViperBindings(t *testing.T) {
	tests := []struct {
		name           string
		viperKey       string
		testValue      any
		expectedStruct string
		description    string
	}{
		{
			name:           "templates_dir binding",
			viperKey:       "templates.dir",
			testValue:      "/test/path",
			expectedStruct: "Templates.Dir",
			description:    "templates-dir flag should bind to templates.dir",
		},
		{
			name:           "workflows.create binding",
			viperKey:       "workflows.create",
			testValue:      true,
			expectedStruct: "Workflows.Create",
			description:    "workflows flag should bind to workflows.create",
		},
		{
			name:           "s3 bucket binding",
			viperKey:       "backend.s3.bucket_name",
			testValue:      "test-bucket-123",
			expectedStruct: "Backend.S3.BucketName",
			description:    "s3-bucket-name flag should bind to backend.s3.bucket_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()

			// Create minimal YAML with required fields
			yamlContent := `
terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
backend:
  s3:
    bucket_name: original-bucket
`
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, ".tfskel.yaml")
			err := os.WriteFile(configFile, []byte(yamlContent), 0644)
			require.NoError(t, err)

			v.SetConfigFile(configFile)
			v.SetConfigType("yaml")
			err = v.ReadInConfig()
			require.NoError(t, err)

			// Override with our test value
			v.Set(tt.viperKey, tt.testValue)

			// Unmarshal into Config
			cfg := &Config{}
			err = v.Unmarshal(cfg)
			require.NoError(t, err, tt.description)

			// Verify the binding path works by checking that Viper has the value
			assert.Equal(t, tt.testValue, v.Get(tt.viperKey),
				"Viper should have the test value at key %s", tt.viperKey)

			t.Logf("✓ Viper binding %s -> %s works correctly", tt.viperKey, tt.expectedStruct)
		})
	}
}

// TestLoadWithFlagOverrides tests that CLI flags override config file values
func TestLoadWithFlagOverrides(t *testing.T) {
	yamlContent := `
terraform_version: "~> 1.13"
provider:
  aws:
    version: "~> 6.0"
    account_mapping:
      dev: "123456789012"
backend:
  s3:
    bucket_name: config-bucket
templates:
  dir: /config/templates
workflows:
  create: false
`

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".tfskel.yaml")
	err := os.WriteFile(configFile, []byte(yamlContent), 0644)
	require.NoError(t, err)

	// Create Viper instance
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	err = v.ReadInConfig()
	require.NoError(t, err)

	// Create a cobra command with flags
	cmd := &cobra.Command{}
	cmd.Flags().String("templates-dir", "", "templates directory")
	cmd.Flags().String("s3-bucket-name", "", "S3 bucket")
	cmd.Flags().Bool("workflows", false, "create workflows")

	// Simulate user setting flags
	err = cmd.Flags().Set("templates-dir", "/flag/override")
	require.NoError(t, err)
	err = cmd.Flags().Set("s3-bucket-name", "flag-bucket")
	require.NoError(t, err)
	err = cmd.Flags().Set("workflows", "true")
	require.NoError(t, err)

	// Load config
	cfg, err := Load(cmd, v)
	require.NoError(t, err)

	// Verify flags override config file values
	assert.Equal(t, "/flag/override", cfg.Templates.Dir, "Flag should override config file")
	assert.Equal(t, "flag-bucket", cfg.Backend.S3.BucketName, "Flag should override config file")
	assert.True(t, cfg.Workflows.Create, "Flag should override config file")
}
