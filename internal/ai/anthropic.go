package ai

import (
	"context"
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

// AnthropicClient is the Claude implementation of Client.
type AnthropicClient struct {
	client anthropic.Client
	model  string
	maxTok int64
}

// NewAnthropicClient builds a Client backed by Anthropic's Messages API.
// Returns ErrMissingAPIKey when the env var is empty so the caller can warn
// without constructing a doomed request.
func NewAnthropicClient(cfg *Config) (*AnthropicClient, error) {
	key := os.Getenv(APIKeyEnvVar)
	if key == "" {
		return nil, ErrMissingAPIKey
	}
	c := anthropic.NewClient(option.WithAPIKey(key))
	return &AnthropicClient{
		client: c,
		model:  cfg.Model,
		maxTok: int64(cfg.MaxTokens),
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
	body, err := payload.MarshalCompact()
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
