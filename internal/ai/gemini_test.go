package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewGeminiClient_MissingKey ensures the constructor returns
// ErrMissingGeminiAPIKey when GEMINI_API_KEY is unset. Callers depend on this
// sentinel to decide between "warn and skip" vs propagating an error.
func TestNewGeminiClient_MissingKey(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "")
	cfg := &Config{Model: GeminiDefaultModel, MaxTokens: DefaultMaxTokens}
	client, err := NewGeminiClient(context.Background(), cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, errors.Is(err, ErrMissingGeminiAPIKey),
		"expected ErrMissingGeminiAPIKey sentinel, got %v", err)
}

func TestNewGeminiClient_WithKey(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "test-key-not-used-in-this-test")
	cfg := &Config{Model: GeminiDefaultModel, MaxTokens: DefaultMaxTokens}
	client, err := NewGeminiClient(context.Background(), cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, GeminiDefaultModel, client.model)
	assert.Equal(t, int32(DefaultMaxTokens), client.maxTok)
}

// TestNewGeminiClient_RemapsAnthropicDefault verifies that when the caller
// passes the Anthropic DefaultModel (the global default in LoadConfig), the
// Gemini constructor remaps it to GeminiDefaultModel — users running
// TFSKEL_AI_PROVIDER=gemini should not have to also set --ai-model.
func TestNewGeminiClient_RemapsAnthropicDefault(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "test-key-not-used-in-this-test")
	cfg := &Config{Model: DefaultModel, MaxTokens: DefaultMaxTokens}
	client, err := NewGeminiClient(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, GeminiDefaultModel, client.model)
}

// TestNewGeminiClient_RespectsExplicitModel verifies that a user-supplied
// model (via --ai-model) is preserved rather than remapped.
func TestNewGeminiClient_RespectsExplicitModel(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "test-key-not-used-in-this-test")
	const explicit = "gemini-2.5-flash"
	cfg := &Config{Model: explicit, MaxTokens: DefaultMaxTokens}
	client, err := NewGeminiClient(context.Background(), cfg)
	require.NoError(t, err)
	assert.Equal(t, explicit, client.model)
}
