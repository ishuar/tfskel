package ai

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewClient_DefaultsToAnthropic verifies that an unset provider env var
// selects the Anthropic backend — the documented default behavior that
// existing users rely on.
func TestNewClient_DefaultsToAnthropic(t *testing.T) {
	t.Setenv(ProviderEnvVar, "")
	t.Setenv(APIKeyEnvVar, "test-key-not-used-in-this-test")
	t.Setenv(GeminiAPIKeyEnvVar, "")
	cfg := &Config{}
	c, err := NewClient(context.Background(), cfg)
	require.NoError(t, err)
	_, ok := c.(*AnthropicClient)
	assert.True(t, ok, "expected *AnthropicClient, got %T", c)
}

func TestNewClient_ExplicitAnthropic(t *testing.T) {
	t.Setenv(ProviderEnvVar, ProviderAnthropic)
	t.Setenv(APIKeyEnvVar, "test-key-not-used-in-this-test")
	cfg := &Config{}
	c, err := NewClient(context.Background(), cfg)
	require.NoError(t, err)
	_, ok := c.(*AnthropicClient)
	assert.True(t, ok, "expected *AnthropicClient, got %T", c)
}

func TestNewClient_Gemini(t *testing.T) {
	t.Setenv(ProviderEnvVar, ProviderGemini)
	t.Setenv(GeminiAPIKeyEnvVar, "test-key-not-used-in-this-test")
	cfg := &Config{}
	c, err := NewClient(context.Background(), cfg)
	require.NoError(t, err)
	_, ok := c.(*GeminiClient)
	assert.True(t, ok, "expected *GeminiClient, got %T", c)
}

// TestNewClient_UnknownProvider verifies that a typo in TFSKEL_AI_PROVIDER
// surfaces ErrUnknownProvider rather than silently falling back to a default
// — the failure mode users want to see is a clear error, not a wrong-backend
// surprise.
func TestNewClient_UnknownProvider(t *testing.T) {
	t.Setenv(ProviderEnvVar, "openai")
	cfg := &Config{}
	c, err := NewClient(context.Background(), cfg)
	require.Error(t, err)
	assert.Nil(t, c)
	assert.True(t, errors.Is(err, ErrUnknownProvider),
		"expected ErrUnknownProvider sentinel, got %v", err)
}

// TestNewClient_GeminiMissingKey verifies that missing GEMINI_API_KEY
// propagates the sentinel through the factory so the caller can warn-and-skip
// the same way it does for Anthropic.
func TestNewClient_GeminiMissingKey(t *testing.T) {
	t.Setenv(ProviderEnvVar, ProviderGemini)
	t.Setenv(GeminiAPIKeyEnvVar, "")
	cfg := &Config{}
	c, err := NewClient(context.Background(), cfg)
	require.Error(t, err)
	assert.Nil(t, c)
	assert.True(t, errors.Is(err, ErrMissingGeminiAPIKey),
		"expected ErrMissingGeminiAPIKey sentinel, got %v", err)
}
