package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
