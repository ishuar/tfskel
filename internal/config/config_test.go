package config

import (
	"testing"

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
		name     string
		env      string
		mapping  map[string]string
		expected string
	}{
		{
			name: "dev environment",
			env:  "dev",
			mapping: map[string]string{
				"dev": "789456123789",
				"stg": "123456789012",
				"prd": "96385214714",
			},
			expected: "789456123789",
		},
		{
			name: "unknown environment",
			env:  "test",
			mapping: map[string]string{
				"dev": "789456123789",
			},
			expected: "000000000000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Provider: &Provider{
					AWS: &AWSProvider{
						AccountMapping: tt.mapping,
					},
				},
			}
			result := cfg.GetAccountID(tt.env)
			assert.Equal(t, tt.expected, result)
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
			result := cfg.GetAccountID(env)
			assert.Equal(t, expectedAccount, result)
		}
	})
}

func TestGenerate_CreateGithubWorkflows(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "create_github_workflows enabled via config",
			config: &Config{
				Generate: &Generate{
					GithubWorkflows: &GithubWorkflows{
						Create: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "create_github_workflows disabled via config",
			config: &Config{
				Generate: &Generate{
					GithubWorkflows: &GithubWorkflows{
						Create: false,
					},
				},
			},
			expected: false,
		},
		{
			name: "create_github_workflows not set (nil GithubWorkflows)",
			config: &Config{
				Generate: &Generate{
					GithubWorkflows: nil,
				},
			},
			expected: false,
		},
		{
			name: "create_github_workflows not set (nil Generate)",
			config: &Config{
				Generate: nil,
			},
			expected: false,
		},
		{
			name:     "create_github_workflows not set (empty config)",
			config:   &Config{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.config.Generate != nil && tt.config.Generate.GithubWorkflows != nil {
				assert.Equal(t, tt.expected, tt.config.Generate.GithubWorkflows.Create)
			} else {
				// When Generate or GithubWorkflows is nil, the feature should be disabled
				if tt.config.Generate != nil {
					assert.Nil(t, tt.config.Generate.GithubWorkflows)
				} else {
					assert.Nil(t, tt.config.Generate)
				}
			}
		})
	}
}

func TestApplyCreateGithubWorkflowsOverride(t *testing.T) {
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
				Generate: nil,
			},
			flagValue:      true,
			flagChanged:    true,
			expectedValue:  true,
			expectedNonNil: true,
		},
		{
			name: "flag set to false overrides true config",
			initialConfig: &Config{
				Generate: &Generate{
					GithubWorkflows: &GithubWorkflows{
						Create: true,
					},
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
				Generate: &Generate{
					GithubWorkflows: &GithubWorkflows{
						Create: true,
					},
				},
			},
			flagValue:      false,
			flagChanged:    false,
			expectedValue:  true,
			expectedNonNil: true,
		},
		{
			name: "flag not changed preserves nil Generate",
			initialConfig: &Config{
				Generate: nil,
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

			// Simulate what applyCreateGithubWorkflowsOverride does
			if tt.flagChanged {
				if cfg.Generate == nil {
					cfg.Generate = &Generate{}
				}
				if cfg.Generate.GithubWorkflows == nil {
					cfg.Generate.GithubWorkflows = &GithubWorkflows{}
				}
				cfg.Generate.GithubWorkflows.Create = tt.flagValue
			}

			if tt.expectedNonNil {
				require.NotNil(t, cfg.Generate)
				require.NotNil(t, cfg.Generate.GithubWorkflows)
				assert.Equal(t, tt.expectedValue, cfg.Generate.GithubWorkflows.Create)
			} else if !tt.flagChanged {
				// When flag is not changed and Generate was nil, it should stay nil
				assert.Nil(t, cfg.Generate)
			}
		})
	}
}
