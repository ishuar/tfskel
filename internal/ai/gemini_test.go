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

// TestNewGeminiClient_MissingKey ensures the constructor returns
// ErrMissingGeminiAPIKey when GEMINI_API_KEY is unset. Callers depend on this
// sentinel to decide between "warn and skip" vs propagating an error.
func TestNewGeminiClient_MissingKey(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "")
	client, err := NewGeminiClient(context.Background(), &Config{})
	require.Error(t, err)
	assert.Nil(t, client)
	assert.True(t, errors.Is(err, ErrMissingGeminiAPIKey),
		"expected ErrMissingGeminiAPIKey sentinel, got %v", err)
}

// TestNewGeminiClient_Defaults verifies a zero-value Config resolves to this
// provider's own default model — no cross-provider remapping involved:
// users running TFSKEL_AI_PROVIDER=gemini should not have to set --ai-model.
func TestNewGeminiClient_Defaults(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "test-key-not-used-in-this-test")
	client, err := NewGeminiClient(context.Background(), &Config{})
	require.NoError(t, err)
	assert.Equal(t, GeminiDefaultModel, client.Model())
}

// TestNewGeminiClient_RespectsExplicitModel verifies that a user-supplied
// model (via --ai-model) is preserved.
func TestNewGeminiClient_RespectsExplicitModel(t *testing.T) {
	t.Setenv(GeminiAPIKeyEnvVar, "test-key-not-used-in-this-test")
	const explicit = "gemini-2.5-pro"
	client, err := NewGeminiClient(context.Background(), &Config{Model: explicit})
	require.NoError(t, err)
	assert.Equal(t, explicit, client.Model())
}

// geminiSSEChunk is one streamGenerateContent SSE data frame.
type geminiSSEChunk struct {
	text         string
	finishReason string
}

// geminiSSEServer serves a canned streamGenerateContent SSE stream.
func geminiSSEServer(t *testing.T, chunks []geminiSSEChunk) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "streamGenerateContent",
			"expected a streaming request, got %s", r.URL.Path)
		w.Header().Set("Content-Type", "text/event-stream")
		for _, c := range chunks {
			text, err := json.Marshal(c.text)
			require.NoError(t, err)
			finish := ""
			if c.finishReason != "" {
				finish = fmt.Sprintf(`,"finishReason":%q`, c.finishReason)
			}
			fmt.Fprintf(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":%s}],\"role\":\"model\"},\"index\":0%s}]}\n\n", text, finish)
		}
	}))
}

func newTestGeminiClient(t *testing.T, srv *httptest.Server, cfg *Config) *GeminiClient {
	t.Helper()
	t.Setenv(GeminiAPIKeyEnvVar, "test-key")
	client, err := NewGeminiClient(context.Background(), cfg, WithBaseURL(srv.URL), WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	return client
}

func TestGeminiExplain_StreamsText(t *testing.T) {
	srv := geminiSSEServer(t, []geminiSSEChunk{
		{text: "## Blast Radius\n"},
		{text: ""},
		{text: "The database replacement is the highest-risk change.", finishReason: "STOP"},
	})
	defer srv.Close()
	client := newTestGeminiClient(t, srv, &Config{})

	var out strings.Builder
	err := client.Explain(context.Background(), &Payload{}, &out)

	require.NoError(t, err)
	assert.Equal(t, "## Blast Radius\nThe database replacement is the highest-risk change.", out.String(),
		"chunks stream in order; empty chunks are skipped")
}

// TestGeminiExplain_Truncation locks in the ErrResponseTruncated contract for
// the Gemini adapter: a MAX_TOKENS finish reason on any candidate surfaces as
// the sentinel after the partial text was written.
func TestGeminiExplain_Truncation(t *testing.T) {
	srv := geminiSSEServer(t, []geminiSSEChunk{
		{text: "Partial ana", finishReason: "MAX_TOKENS"},
	})
	defer srv.Close()
	client := newTestGeminiClient(t, srv, &Config{})

	var out strings.Builder
	err := client.Explain(context.Background(), &Payload{}, &out)

	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrResponseTruncated),
		"expected ErrResponseTruncated sentinel, got %v", err)
	assert.Contains(t, err.Error(), fmt.Sprintf("max_tokens=%d", DefaultMaxTokens),
		"zero-value MaxTokens resolves to the provider default")
	assert.Equal(t, "Partial ana", out.String(), "partial text is written before the error")
}

func TestGeminiExplain_WriterErrorMidStream(t *testing.T) {
	srv := geminiSSEServer(t, []geminiSSEChunk{{text: "some text", finishReason: "STOP"}})
	defer srv.Close()
	client := newTestGeminiClient(t, srv, &Config{})

	err := client.Explain(context.Background(), &Payload{}, failingWriter{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "write streamed token")
}
