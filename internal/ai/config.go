// Package ai produces a human-readable narrative analysis of a Terraform plan
// by sending a sanitized, compressed view of the plan to Anthropic's Claude API.
//
// The package is layered so additional providers (or an MCP server surface) can
// be added later without touching command code:
//
//   - Client   is the provider-agnostic interface (Explain streams Markdown).
//   - Payload  is the wire-format shipped to the model (built from a parsed plan).
//   - Config   holds model + max-tokens settings (loaded from viper).
//
// v1 ships only the Anthropic implementation. The ANTHROPIC_API_KEY environment
// variable is the sole credential source — keys are never read from config files
// or command-line flags.
package ai

import "github.com/spf13/viper"

// DefaultMaxTokens caps a single response. 8192 fits the three required
// Markdown sections (Blast Radius, Security, Rollback & Pre-apply) with
// headroom for large plans. The previous 4096 default routinely truncated
// mid-sentence — output budgets are cheap on a per-call basis and the only
// real cost of a high ceiling is a slightly longer wall-clock when the model
// decides to use it.
const DefaultMaxTokens = 8192

// Config holds AI-feature settings sourced from viper (file/env) and overridden
// by command-line flags at the call site.
//
// Zero values mean "the provider's choice": an empty Model and a zero
// MaxTokens are resolved by each provider constructor, so one provider's
// default never leaks into another's.
type Config struct {
	Model     string `mapstructure:"model"`
	MaxTokens int    `mapstructure:"max_tokens"`
}

// LoadConfig reads AI settings from viper. Unset (or unusable) fields stay at
// their zero value and are defaulted by the provider constructor.
func LoadConfig(v *viper.Viper) *Config {
	cfg := &Config{}
	if v.IsSet("ai.model") {
		cfg.Model = v.GetString("ai.model")
	}
	if v.IsSet("ai.max_tokens") {
		if n := v.GetInt("ai.max_tokens"); n > 0 {
			cfg.MaxTokens = n
		}
	}
	return cfg
}
