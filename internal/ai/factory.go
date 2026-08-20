package ai

import (
	"context"
	"errors"
	"fmt"
	"os"
)

// ErrUnknownProvider signals that TFSKEL_AI_PROVIDER holds an unrecognized
// value. Returned wrapped with the offending value so callers can surface a
// useful error message without re-parsing the env.
var ErrUnknownProvider = errors.New("unknown AI provider")

// ProviderEnvVar selects the AI backend at runtime. Recognized values are
// "anthropic" (default) and "gemini". The env var is the only way to switch
// providers — keeping it out of config files mirrors how API keys are handled
// and keeps provider choice an explicit, per-invocation decision.
const ProviderEnvVar = "TFSKEL_AI_PROVIDER"

// Recognized values for ProviderEnvVar.
const (
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
)

// NewClient constructs the Client implementation selected by ProviderEnvVar.
// An unrecognized provider value is an error — silent fallback to Anthropic
// would hide typos in the env var.
func NewClient(ctx context.Context, cfg *Config) (Client, error) {
	switch provider := os.Getenv(ProviderEnvVar); provider {
	case "", ProviderAnthropic:
		return NewAnthropicClient(cfg)
	case ProviderGemini:
		return NewGeminiClient(ctx, cfg)
	default:
		return nil, fmt.Errorf("%w: %s=%q (want %q or %q)", ErrUnknownProvider, ProviderEnvVar, provider, ProviderAnthropic, ProviderGemini)
	}
}
