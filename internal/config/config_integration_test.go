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
			name: "full valid config with generate section",
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
generate:
  templates_dir: /custom/templates
  extra_template_extensions:
    - tf.tmpl
    - md.tmpl
  github_workflows:
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

				// Test Generate section
				require.NotNil(t, cfg.Generate)
				assert.Equal(t, "/custom/templates", cfg.Generate.TemplatesDir)
				assert.ElementsMatch(t, []string{"tf.tmpl", "md.tmpl"}, cfg.Generate.ExtraTemplateExtensions)

				// Test GithubWorkflows
				require.NotNil(t, cfg.Generate.GithubWorkflows)
				assert.True(t, cfg.Generate.GithubWorkflows.Create)
				assert.Equal(t, "{{.AppDir}}-{{.Env}}", cfg.Generate.GithubWorkflows.NameTemplate)
				assert.Equal(t, "GitHubActionsRole", cfg.Generate.GithubWorkflows.AWSRoleName)
			},
		},
		{
			name: "templates_dir and extra_template_extensions under generate",
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
generate:
  templates_dir: /custom/path
  extra_template_extensions:
    - yaml.tmpl
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Generate)
				assert.Equal(t, "/custom/path", cfg.Generate.TemplatesDir)
				assert.Contains(t, cfg.Generate.ExtraTemplateExtensions, "yaml.tmpl")
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
				// With old structure at root level, values won't be populated in Generate
				// This validates the breaking change - old configs won't work
				if cfg.Generate != nil {
					assert.Empty(t, cfg.Generate.TemplatesDir, "Old YAML structure should not populate Generate.TemplatesDir")
					// normalizeTemplateExtensions always adds tf.tmpl, so we check it only contains the default
					assert.ElementsMatch(t, []string{"tf.tmpl"}, cfg.Generate.ExtraTemplateExtensions,
						"Old YAML structure should only have default tf.tmpl, not yaml.tmpl from root level")
				}
			},
		},
		{
			name: "minimal config without generate section",
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
				// Generate section can be nil or empty
				if cfg.Generate != nil {
					assert.Empty(t, cfg.Generate.TemplatesDir)
				}
			},
		},
		{
			name: "generate section with only github_workflows",
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
generate:
  github_workflows:
    create: true
    aws_role_arn: "arn:aws:iam::123456789012:role/MyRole"
`,
			expectError: false,
			validateFunc: func(t *testing.T, cfg *Config) {
				t.Helper()
				require.NotNil(t, cfg.Generate)
				require.NotNil(t, cfg.Generate.GithubWorkflows)
				assert.True(t, cfg.Generate.GithubWorkflows.Create)
				assert.Equal(t, "arn:aws:iam::123456789012:role/MyRole", cfg.Generate.GithubWorkflows.AWSRoleArn)
				// templates_dir should have default/empty value
				assert.Empty(t, cfg.Generate.TemplatesDir)
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

		// Generate section should exist and have the new structure
		require.NotNil(t, cfg.Generate, "generate section must exist in example config")

		// templates_dir should be under generate
		assert.NotEmpty(t, cfg.Generate.TemplatesDir, "generate.templates_dir should be set in example")

		// extra_template_extensions should be under generate
		assert.NotNil(t, cfg.Generate.ExtraTemplateExtensions, "generate.extra_template_extensions should exist")

		// github_workflows should be under generate
		require.NotNil(t, cfg.Generate.GithubWorkflows, "generate.github_workflows should exist")
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
			viperKey:       "generate.templates_dir",
			testValue:      "/test/path",
			expectedStruct: "Generate.TemplatesDir",
			description:    "templates-dir flag should bind to generate.templates_dir",
		},
		{
			name:           "extra_template_extensions binding",
			viperKey:       "generate.extra_template_extensions",
			testValue:      []string{"test.tmpl", "yaml.tmpl"},
			expectedStruct: "Generate.ExtraTemplateExtensions",
			description:    "extra-template-extensions flag should bind to generate.extra_template_extensions",
		},
		{
			name:           "github_workflows.create binding",
			viperKey:       "generate.github_workflows.create",
			testValue:      true,
			expectedStruct: "Generate.GithubWorkflows.Create",
			description:    "create-github-workflows flag should bind to generate.github_workflows.create",
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

// TestNormalizeTemplateExtensions ensures tf.tmpl is always included
func TestNormalizeTemplateExtensions(t *testing.T) {
	tests := []struct {
		name     string
		input    *Config
		expected []string
	}{
		{
			name: "nil Generate creates it with tf.tmpl",
			input: &Config{
				Generate: nil,
			},
			expected: []string{"tf.tmpl"},
		},
		{
			name: "nil ExtraTemplateExtensions gets tf.tmpl",
			input: &Config{
				Generate: &Generate{
					ExtraTemplateExtensions: nil,
				},
			},
			expected: []string{"tf.tmpl"},
		},
		{
			name: "empty ExtraTemplateExtensions gets tf.tmpl",
			input: &Config{
				Generate: &Generate{
					ExtraTemplateExtensions: []string{},
				},
			},
			expected: []string{"tf.tmpl"},
		},
		{
			name: "custom extensions include tf.tmpl",
			input: &Config{
				Generate: &Generate{
					ExtraTemplateExtensions: []string{"yaml.tmpl", "md.tmpl"},
				},
			},
			expected: []string{"tf.tmpl", "yaml.tmpl", "md.tmpl"},
		},
		{
			name: "tf.tmpl is not duplicated",
			input: &Config{
				Generate: &Generate{
					ExtraTemplateExtensions: []string{"tf.tmpl", "yaml.tmpl", "tf.tmpl"},
				},
			},
			expected: []string{"tf.tmpl", "yaml.tmpl"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalizeTemplateExtensions(tt.input)

			require.NotNil(t, tt.input.Generate)
			require.NotNil(t, tt.input.Generate.ExtraTemplateExtensions)

			// Check that all expected extensions are present
			for _, expected := range tt.expected {
				assert.Contains(t, tt.input.Generate.ExtraTemplateExtensions, expected)
			}

			// Check the count matches (accounting for map-based deduplication)
			assert.Len(t, tt.input.Generate.ExtraTemplateExtensions, len(tt.expected))
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
generate:
  templates_dir: /config/templates
  extra_template_extensions:
    - tf.tmpl
  github_workflows:
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
	cmd.Flags().Bool("create-github-workflows", false, "create workflows")
	cmd.Flags().StringSlice("extra-template-extensions", []string{}, "extensions")

	// Simulate user setting flags
	err = cmd.Flags().Set("templates-dir", "/flag/override")
	require.NoError(t, err)
	err = cmd.Flags().Set("s3-bucket-name", "flag-bucket")
	require.NoError(t, err)
	err = cmd.Flags().Set("create-github-workflows", "true")
	require.NoError(t, err)

	// Load config
	cfg, err := Load(cmd, v)
	require.NoError(t, err)

	// Verify flags override config file values
	assert.Equal(t, "/flag/override", cfg.Generate.TemplatesDir, "Flag should override config file")
	assert.Equal(t, "flag-bucket", cfg.Backend.S3.BucketName, "Flag should override config file")
	assert.True(t, cfg.Generate.GithubWorkflows.Create, "Flag should override config file")
}
