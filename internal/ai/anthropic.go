package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// APIKeyEnvVar is the sole credential source for the Anthropic client.
// The key is never read from config or flags — see package doc.
//
//nolint:gosec // G101: this is the env var name, not the key itself.
const APIKeyEnvVar = "ANTHROPIC_API_KEY"

// ErrMissingAPIKey signals that the env var is unset. Callers warn and skip
// AI analysis when they see this; they do not propagate it as a command error.
var ErrMissingAPIKey = errors.New("ANTHROPIC_API_KEY is not set")

// AnthropicDefaultModel is the Claude model used when no override is
// configured. Sonnet 4.6 balances capability, latency, and cost for
// plan-narrative work.
const AnthropicDefaultModel = "claude-sonnet-4-6"

// AnthropicClient is the Claude implementation of Client.
type AnthropicClient struct {
	client anthropic.Client
	model  string
	maxTok int64
}

// NewAnthropicClient builds a Client backed by Anthropic's Messages API.
// Returns ErrMissingAPIKey when the env var is empty so the caller can warn
// without constructing a doomed request. Zero-value Config fields resolve to
// this provider's defaults; opts open the transport seam for tests.
func NewAnthropicClient(cfg *Config, opts ...Option) (*AnthropicClient, error) {
	key := os.Getenv(APIKeyEnvVar)
	if key == "" {
		return nil, ErrMissingAPIKey
	}
	o := applyOptions(opts)
	reqOpts := []option.RequestOption{option.WithAPIKey(key)}
	if o.baseURL != "" {
		reqOpts = append(reqOpts, option.WithBaseURL(o.baseURL))
	}
	if o.httpClient != nil {
		reqOpts = append(reqOpts, option.WithHTTPClient(o.httpClient))
	}

	model := cfg.Model
	if model == "" {
		model = AnthropicDefaultModel
	}
	maxTok := cfg.MaxTokens
	if maxTok <= 0 {
		maxTok = DefaultMaxTokens
	}

	return &AnthropicClient{
		client: anthropic.NewClient(reqOpts...),
		model:  model,
		maxTok: int64(maxTok),
	}, nil
}

// Provider identifies the backend in user-facing output.
func (a *AnthropicClient) Provider() string { return ProviderAnthropic }

// Model returns the resolved Claude model name.
func (a *AnthropicClient) Model() string { return a.model }

// Explain streams Claude's narrative analysis to writer as it arrives.
// The system prompt is marked for ephemeral caching so repeated invocations
// within the cache window are billed at the reduced rate.
func (a *AnthropicClient) Explain(ctx context.Context, payload *Payload, writer io.Writer) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: a.maxTok,
		System: []anthropic.TextBlockParam{{
			Text:         SystemPrompt,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Messages: []anthropic.MessageParam{{
			Role: anthropic.MessageParamRoleUser,
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: string(body)},
			}},
		}},
	})

	var stopReason anthropic.StopReason
	for stream.Next() {
		event := stream.Current()
		switch e := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			text := e.Delta.AsTextDelta()
			if text.Text == "" {
				continue
			}
			if _, err := io.WriteString(writer, text.Text); err != nil {
				return fmt.Errorf("write streamed token: %w", err)
			}
		case anthropic.MessageDeltaEvent:
			// The final stop_reason arrives on the message_delta event near the
			// end of the stream; capture it so we can detect max_tokens hits.
			if e.Delta.StopReason != "" {
				stopReason = e.Delta.StopReason
			}
		}
	}
	if err := stream.Err(); err != nil {
		return fmt.Errorf("anthropic stream: %w", err)
	}
	if stopReason == anthropic.StopReasonMaxTokens {
		return fmt.Errorf("%w (provider=anthropic, max_tokens=%d)", ErrResponseTruncated, a.maxTok)
	}
	return nil
}
