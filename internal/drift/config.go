package drift

import "github.com/spf13/viper"

// OutputFormat defines the output format type
type OutputFormat string

// FormatTable, FormatJSON, and FormatCSV are the supported output formats
const (
	FormatTable OutputFormat = "table"
	FormatJSON  OutputFormat = "json"
	FormatCSV   OutputFormat = "csv"

	// Terminal and table width constants
	defaultTerminalWidth = 120 // Default width when terminal size cannot be detected
	minDriftTableWidth   = 113 // File(40) + Type(16) + Expected(16) + Actual(16) + Status(15) + borders(10)
	minPlanTableWidth    = 80  // Minimum width for plan analysis tables
	maxPlanTableWidth    = 150 // Maximum width for readability

	// Width calculation constants
	percentageWidthFactor = 95  // Percentage of terminal width to use
	percentageDivisor     = 100 // Divisor for percentage calculations
	tableBorderPadding    = 4   // Approximate characters needed for borders and padding
	pathDivisor           = 2   // Divisor for calculating extra space from path length

	// File path display constants
	baseFilePathWidth     = 40 // Base file path width before adding extra space
	maxExtraSpaceForPaths = 30 // Maximum extra space to add for long paths

	// Summary display constants
	defaultTopNCount  = 10 // Default number of items to show in top-N summaries
	severityTopNCount = 0  // Show all severity items (0 = no limit)
)

// DriftConfig holds drift-specific configuration
type DriftConfig struct {
	CriticalResources []string `mapstructure:"critical_resources"`
	TopNCount         int      `mapstructure:"top_n_count"`
}

// LoadDriftConfig loads drift configuration from viper.
// Returns a config with user-defined critical resources, or empty list if not configured.
func LoadDriftConfig(v *viper.Viper) *DriftConfig {
	cfg := &DriftConfig{
		TopNCount: defaultTopNCount, // Default to 10
	}

	// Check if the key exists in config
	if v.IsSet("critical_resources") {
		cfg.CriticalResources = v.GetStringSlice("critical_resources")
	}

	// Check if top_n_count is configured
	if v.IsSet("top_n_count") {
		if topN := v.GetInt("top_n_count"); topN > 0 {
			cfg.TopNCount = topN
		}
	}

	return cfg
}
