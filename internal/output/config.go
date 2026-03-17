package output

import "github.com/spf13/viper"

// OutputFormat defines the output format type
type OutputFormat string

// FormatTable, FormatJSON, and FormatCSV are the supported output formats
const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"
)

const (
	// DefaultTerminalWidth is the default width when terminal size cannot be detected
	DefaultTerminalWidth = 120
	// MinDriftTableWidth is the minimum width for drift tables: File(40) + Type(16) + Expected(16) + Actual(16) + Status(15) + borders(10)
	MinDriftTableWidth = 113
	// MinPlanTableWidth is the minimum width for plan analysis tables
	MinPlanTableWidth = 80
	// MaxPlanTableWidth is the maximum width for readability
	MaxPlanTableWidth = 150
)

const (
	// PercentageWidthFactor is the percentage of terminal width to use
	PercentageWidthFactor = 95
	// PercentageDivisor is the divisor for percentage calculations
	PercentageDivisor = 100
	// TableBorderPadding is the approximate characters needed for borders and padding
	TableBorderPadding = 4
	// PathDivisor is the divisor for calculating extra space from path length
	PathDivisor = 2
)

const (
	// BaseFilePathWidth is the base file path width before adding extra space
	BaseFilePathWidth = 40
	// MaxExtraSpaceForPaths is the maximum extra space to add for long paths
	MaxExtraSpaceForPaths = 30
)

const (
	// DefaultTopResourcesCount is the default number of resources to show in top-N summaries
	DefaultTopResourcesCount = 10
	// SeverityTopResourcesCount shows all severity items (0 = no limit)
	SeverityTopResourcesCount = 0
)

// DriftConfig holds drift-specific configuration
type DriftConfig struct {
	CriticalResources []string `mapstructure:"critical_resources"`
	TopResourcesCount int      `mapstructure:"top_resources_count"`
}

// LoadDriftConfig loads drift configuration from viper.
// Returns a config with user-defined critical resources, or empty list if not configured.
func LoadDriftConfig(v *viper.Viper) *DriftConfig {
	cfg := &DriftConfig{
		TopResourcesCount: DefaultTopResourcesCount, // Default to 10
	}

	// Check if the key exists in config
	if v.IsSet("critical_resources") {
		cfg.CriticalResources = v.GetStringSlice("critical_resources")
	}

	// Check if top_resources_count is configured
	if v.IsSet("top_resources_count") {
		if topN := v.GetInt("top_resources_count"); topN > 0 {
			cfg.TopResourcesCount = topN
		}
	}

	return cfg
}
