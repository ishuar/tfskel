package ai

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAnthropicClient_MissingKey ensures the constructor returns
// ErrMissingAPIKey when ANTHROPIC_API_KEY is unset. Callers depend on this
// sentinel to decide between "warn and skip" vs propagating an error.
func TestNewAnthropicClient_MissingKey(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "")
	cfg := &Config{Model: DefaultModel, MaxTokens: DefaultMaxTokens}
	client, err := NewAnthropicClient(cfg)
	require.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, errors.Is(err, ErrMissingAPIKey),
		"expected ErrMissingAPIKey sentinel, got %v", err)
}

func TestNewAnthropicClient_WithKey(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "test-key-not-used-in-this-test")
	cfg := &Config{Model: DefaultModel, MaxTokens: DefaultMaxTokens}
	client, err := NewAnthropicClient(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.Equal(t, DefaultModel, client.model)
	assert.Equal(t, int64(DefaultMaxTokens), client.maxTok)
}
