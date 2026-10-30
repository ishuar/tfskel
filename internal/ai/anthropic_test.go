package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewAnthropicClient_MissingKey ensures the constructor returns
// ErrMissingAPIKey when ANTHROPIC_API_KEY is unset. Callers depend on this
// sentinel to decide between "warn and skip" vs propagating an error.
func TestNewAnthropicClient_MissingKey(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "")
	client, err := NewAnthropicClient(&Config{})
	require.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, errors.Is(err, ErrMissingAPIKey),
		"expected ErrMissingAPIKey sentinel, got %v", err)
}

// TestNewAnthropicClient_Defaults verifies zero-value Config fields resolve
// to this provider's defaults — the provider owns its default, not LoadConfig.
func TestNewAnthropicClient_Defaults(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "test-key-not-used-in-this-test")
	client, err := NewAnthropicClient(&Config{})
	require.NoError(t, err)
	assert.Equal(t, AnthropicDefaultModel, client.Model())
}

func TestNewAnthropicClient_ExplicitModel(t *testing.T) {
	t.Setenv(APIKeyEnvVar, "test-key-not-used-in-this-test")
	client, err := NewAnthropicClient(&Config{Model: "claude-opus-4-7"})
	require.NoError(t, err)
	assert.Equal(t, "claude-opus-4-7", client.Model())
}

// anthropicSSEServer serves a canned Messages-API SSE stream: the given text
// deltas (empty strings included, to exercise the empty-delta skip) followed
// by a message_delta carrying stopReason.
func anthropicSSEServer(t *testing.T, stopReason string, deltas []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/messages"),
			"expected a Messages API request, got %s", r.URL.Path)
		w.Header().Set("Content-Type", "text/event-stream")
		writeEvent := func(event, data string) {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
		}
		writeEvent("message_start", `{"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","model":"claude-test","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":0}}}`)
		writeEvent("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		for _, d := range deltas {
			text, err := json.Marshal(d)
			require.NoError(t, err)
			writeEvent("content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%s}}`, text))
		}
		writeEvent("content_block_stop", `{"type":"content_block_stop","index":0}`)
		writeEvent("message_delta", fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":%q,"stop_sequence":null},"usage":{"output_tokens":5}}`, stopReason))
		writeEvent("message_stop", `{"type":"message_stop"}`)
	}))
}

func newTestAnthropicClient(t *testing.T, srv *httptest.Server, cfg *Config) *AnthropicClient {
	t.Helper()
	t.Setenv(APIKeyEnvVar, "test-key")
	client, err := NewAnthropicClient(cfg, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	return client
}

func TestAnthropicExplain_StreamsText(t *testing.T) {
	srv := anthropicSSEServer(t, "end_turn", []string{"## Blast Radius\n", "", "The database replacement ", "is the highest-risk change."})
	defer srv.Close()
	client := newTestAnthropicClient(t, srv, &Config{})

	var out strings.Builder
	err := client.Explain(context.Background(), &Payload{}, &out)

	require.NoError(t, err)
	assert.Equal(t, "## Blast Radius\nThe database replacement is the highest-risk change.", out.String(),
		"deltas stream in order; empty deltas are skipped")
}

// TestAnthropicExplain_Truncation locks in the ErrResponseTruncated contract:
// a max_tokens stop_reason surfaces as the sentinel after the partial text
// was already written, so the caller can warn without losing the output.
func TestAnthropicExplain_Truncation(t *testing.T) {
	srv := anthropicSSEServer(t, "max_tokens", []string{"Partial ana"})
	defer srv.Close()
	client := newTestAnthropicClient(t, srv, &Config{})

	var out strings.Builder
	err := client.Explain(context.Background(), &Payload{}, &out)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrResponseTruncated),
		"expected ErrResponseTruncated sentinel, got %v", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("max_tokens=%d", DefaultMaxTokens),
		"zero-value MaxTokens resolves to the provider default")
	assert.Equal(t, "Partial ana", out.String(), "partial text is written before the error")
}

// failingWriter fails on the first write, simulating a broken pipe mid-stream.
type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("pipe closed") }

func TestAnthropicExplain_WriterErrorMidStream(t *testing.T) {
	srv := anthropicSSEServer(t, "end_turn", []string{"some text"})
	defer srv.Close()
	client := newTestAnthropicClient(t, srv, &Config{})

	err := client.Explain(context.Background(), &Payload{}, failingWriter{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write streamed token")
}
