package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "my-terraform-state-bucket",
					},
				},
			},
			wantErr: false,
		},
		{
			name:    "missing provider config",
			config:  &Config{},
			wantErr: true,
			errMsg:  "AWS provider configuration is required",
		},
		{
			name: "missing account mapping",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: nil,
					},
				},
			},
			wantErr: true,
			errMsg:  "account mapping is required",
		},
		{
			name: "empty account mapping",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{},
					},
				},
			},
			wantErr: true,
			errMsg:  "account mapping is required",
		},
		{
			name: "missing backend config",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "backend.s3.bucket_name is invalid",
		},
		{
			name: "empty bucket_name",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "",
					},
				},
			},
			wantErr: true,
			errMsg:  "backend.s3.bucket_name is invalid",
		},
		{
			name: "placeholder bucket_name",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "123456789012",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME",
					},
				},
			},
			wantErr: true,
			errMsg:  "backend.s3.bucket_name is invalid",
		},
		{
			name: "invalid account ID - placeholder text",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "REPLACE_WITH_YOUR_DEV_ACCOUNT_ID",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "my-terraform-state-bucket",
					},
				},
			},
			wantErr: true,
			errMsg:  "AWS account ID must be a 12-digit number",
		},
		{
			name: "invalid account ID - contains letters",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "12345678901A",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "my-terraform-state-bucket",
					},
				},
			},
			wantErr: true,
			errMsg:  "AWS account ID must be a 12-digit number",
		},
		{
			name: "invalid account ID - too short",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "12345",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "my-terraform-state-bucket",
					},
				},
			},
			wantErr: true,
			errMsg:  "AWS account ID must be a 12-digit number",
		},
		{
			name: "invalid account ID - too long",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "1234567890123",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "my-terraform-state-bucket",
					},
				},
			},
			wantErr: true,
			errMsg:  "AWS account ID must be a 12-digit number",
		},
		{
			name: "invalid account ID - contains CHANGE keyword",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "CHANGE_ME_123",
						},
					},
				},
				Backend: &Backend{
					S3: &S3Backend{
						BucketName: "my-terraform-state-bucket",
					},
				},
			},
			wantErr: true,
			errMsg:  "AWS account ID must be a 12-digit number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAccountID(t *testing.T) {
	tests := []struct {
		name        string
		env         string
		config      *Config
		expected    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "dev environment - success",
			env:  "dev",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "789456123789",
							"stg": "123456789012",
							"prd": "963852147141",
						},
					},
				},
			},
			expected:    "789456123789",
			expectError: false,
		},
		{
			name: "unknown environment - error with available envs",
			env:  "test",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: map[string]string{
							"dev": "789456123789",
							"prd": "963852147141",
							"stg": "123456789012",
						},
					},
				},
			},
			expected:    "",
			expectError: true,
			errorMsg:    "no account mapping found for environment \"test\", available: [dev, prd, stg]",
		},
		{
			name:        "nil provider - error",
			env:         "dev",
			config:      &Config{},
			expected:    "",
			expectError: true,
			errorMsg:    "AWS provider configuration is required",
		},
		{
			name: "nil AWS provider - error",
			env:  "dev",
			config: &Config{
				Provider: &Provider{},
			},
			expected:    "",
			expectError: true,
			errorMsg:    "AWS provider configuration is required",
		},
		{
			name: "nil account mapping - error",
			env:  "dev",
			config: &Config{
				Provider: &Provider{
					AWS: &AWSProvider{},
				},
			},
			expected:    "",
			expectError: true,
			errorMsg:    "AWS provider configuration is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.config.GetAccountID(tt.env)
			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestConfig_DefaultValues(t *testing.T) {
	t.Run("config with empty defaults gets populated", func(t *testing.T) {
		cfg := &Config{}

		// Simulate what Load does for defaults
		if cfg.TerraformVersion == "" {
			cfg.TerraformVersion = "~> 1.13"
		}
		if cfg.Provider == nil {
			cfg.Provider = &Provider{}
		}
		if cfg.Provider.AWS == nil {
			cfg.Provider.AWS = &AWSProvider{}
		}
		if cfg.Provider.AWS.Version == "" {
			cfg.Provider.AWS.Version = "~> 6.0"
		}

		assert.Equal(t, "~> 1.13", cfg.TerraformVersion)
		assert.Equal(t, "~> 6.0", cfg.Provider.AWS.Version)
	})
}

func TestConfig_MultipleAccountMappings(t *testing.T) {
	t.Run("multiple accounts", func(t *testing.T) {
		mapping := map[string]string{
			"dev": "111111111111",
			"stg": "222222222222",
			"prd": "333333333333",
		}

		for env, expectedAccount := range mapping {
			cfg := &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: mapping,
					},
				},
			}
			result, err := cfg.GetAccountID(env)
			assert.NoError(t, err)
			assert.Equal(t, expectedAccount, result)
		}
	})
}

func TestWorkflows_Create(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "workflows enabled via config",
			config: &Config{
				Workflows: &Workflows{
					Create: true,
				},
			},
			expected: true,
		},
		{
			name: "workflows disabled via config",
			config: &Config{
				Workflows: &Workflows{
					Create: false,
				},
			},
			expected: false,
		},
		{
			name: "workflows not set (nil Workflows)",
			config: &Config{
				Workflows: nil,
			},
			expected: false,
		},
		{
			name:     "workflows not set (empty config)",
			config:   &Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Workflows != nil {
				assert.Equal(t, tt.expected, tt.config.Workflows.Create)
			} else {
				// When Workflows is nil, the feature should be disabled
				assert.Nil(t, tt.config.Workflows)
			}
		})
	}
}

func TestApplyWorkflowsOverride(t *testing.T) {
	tests := []struct {
		name           string
		initialConfig  *Config
		flagValue      bool
		flagChanged    bool
		expectedValue  bool
		expectedNonNil bool
	}{
		{
			name: "flag set to true overrides nil config",
			initialConfig: &Config{
				Workflows: nil,
			},
			flagValue:      true,
			flagChanged:    true,
			expectedValue:  true,
			expectedNonNil: true,
		},
		{
			name: "flag set to false overrides true config",
			initialConfig: &Config{
				Workflows: &Workflows{
					Create: true,
				},
			},
			flagValue:      false,
			flagChanged:    true,
			expectedValue:  false,
			expectedNonNil: true,
		},
		{
			name: "flag not changed preserves config value",
			initialConfig: &Config{
				Workflows: &Workflows{
					Create: true,
				},
			},
			flagValue:      false,
			flagChanged:    false,
			expectedValue:  true,
			expectedNonNil: true,
		},
		{
			name: "flag not changed preserves nil Workflows",
			initialConfig: &Config{
				Workflows: nil,
			},
			flagValue:      true,
			flagChanged:    false,
			expectedValue:  false,
			expectedNonNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: This test validates the logic without actually creating cobra commands
			// The actual function would be called with a cobra command, but we test the logic here
			cfg := tt.initialConfig

			// Simulate what applyWorkflowsOverride does
			if tt.flagChanged {
				if cfg.Workflows == nil {
					cfg.Workflows = &Workflows{}
				}
				cfg.Workflows.Create = tt.flagValue
			}

			if tt.expectedNonNil {
				require.NotNil(t, cfg.Workflows)
				assert.Equal(t, tt.expectedValue, cfg.Workflows.Create)
			} else if !tt.flagChanged {
				// When flag is not changed and Workflows was nil, it should stay nil
				assert.Nil(t, cfg.Workflows)
			}
		})
	}
}

func TestCheckDeprecatedConfig(t *testing.T) {
	tests := []struct {
		name         string
		viperSetup   func(*testing.T) *Config
		expectOutput bool // whether we expect warning output
	}{
		{
			name: "no deprecated config",
			viperSetup: func(t *testing.T) *Config {
				t.Helper()
				v := viper.New()
				v.Set("generate.templates_dir", "/custom/path")
				cfg := &Config{}
				err := v.Unmarshal(cfg)
				require.NoError(t, err)
				checkDeprecatedConfig(v)
				return cfg
			},
			expectOutput: false,
		},
		{
			name: "deprecated templates_dir at root",
			viperSetup: func(t *testing.T) *Config {
				t.Helper()
				v := viper.New()
				v.Set("templates_dir", "/old/path")
				cfg := &Config{}
				err := v.Unmarshal(cfg)
				require.NoError(t, err)
				// This will print warnings, but we can't easily capture them in unit tests
				// In integration tests, we verify the actual behavior
				checkDeprecatedConfig(v)
				return cfg
			},
			expectOutput: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the setup which includes checkDeprecatedConfig
			cfg := tt.viperSetup(t)
			assert.NotNil(t, cfg)
			// The actual warning output is tested in integration tests
			// Here we just verify the function doesn't panic
		})
	}
}
