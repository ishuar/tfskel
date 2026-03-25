package plan

import "github.com/spf13/viper"

const (
	// DefaultTopResourcesCount is the default number of resources to show in top-N summaries
	DefaultTopResourcesCount = 10
)

// AnalysisConfig holds plan analysis-specific configuration
type AnalysisConfig struct {
	CriticalResources []string `mapstructure:"critical_resources"`
	TopResourcesCount int      `mapstructure:"top_resources_count"`
}

// LoadAnalysisConfig loads plan analysis configuration from viper.
// Returns a config with user-defined critical resources, or empty list if not configured.
func LoadAnalysisConfig(v *viper.Viper) *AnalysisConfig {
	cfg := &AnalysisConfig{
		TopResourcesCount: DefaultTopResourcesCount, // Default to 10
	}

	// Check if the key exists in config
	if v.IsSet("critical_resources") {
		cfg.CriticalResources = v.GetStringSlice("critical_resources")
	}

	// Check if top_resources_count is configured
	// 0 = show all (unlimited), negative = use default (10), positive = use that limit
	if v.IsSet("top_resources_count") {
		if topN := v.GetInt("top_resources_count"); topN >= 0 {
			cfg.TopResourcesCount = topN
		}
	}

	return cfg
}
