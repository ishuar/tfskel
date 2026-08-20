package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"google.golang.org/genai"
)

// GeminiAPIKeyEnvVar is the sole credential source for the Gemini client.
// GEMINI_API_KEY is the canonical env var name documented by Google AI Studio.
//
//nolint:gosec // G101: this is the env var name, not the key itself.
const GeminiAPIKeyEnvVar = "GEMINI_API_KEY"

// GeminiDefaultModel is the Gemini model used when no override is configured.
// 2.5 Flash is selected for broad free-tier availability; 2.5 Pro free-tier
// access is restricted (projects often see limit: 0) and not safe as a default.
const GeminiDefaultModel = "gemini-2.5-flash"

// ErrMissingGeminiAPIKey signals that GEMINI_API_KEY is unset. Callers warn and
// skip AI analysis when they see this; they do not propagate it as a command error.
var ErrMissingGeminiAPIKey = errors.New("GEMINI_API_KEY is not set")

// GeminiClient is the Gemini implementation of Client.
type GeminiClient struct {
	client *genai.Client
	model  string
	maxTok int32
}

// NewGeminiClient builds a Client backed by Google's Gemini API.
// Returns ErrMissingGeminiAPIKey when the env var is empty so the caller can
// warn without constructing a doomed request.
func NewGeminiClient(ctx context.Context, cfg *Config) (*GeminiClient, error) {
	key := os.Getenv(GeminiAPIKeyEnvVar)
	if key == "" {
		return nil, ErrMissingGeminiAPIKey
	}
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  key,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("init gemini client: %w", err)
	}
	model := cfg.Model
	if model == "" || model == DefaultModel {
		model = GeminiDefaultModel
	}
	return &GeminiClient{
		client: c,
		model:  model,
		//nolint:gosec // G115: cfg.MaxTokens is loaded from viper with a 4096 default; values above int32 max are nonsensical for a single response and would be rejected by the API.
		maxTok: int32(cfg.MaxTokens),
	}, nil
}

// Provider identifies the backend in user-facing output.
func (g *GeminiClient) Provider() string { return ProviderGemini }

// Model returns the resolved Gemini model name.
func (g *GeminiClient) Model() string { return g.model }

// Explain streams Gemini's narrative analysis to writer as it arrives.
// The system prompt is passed via SystemInstruction so it is not re-tokenized
// as part of the user message on each call.
func (g *GeminiClient) Explain(ctx context.Context, payload *Payload, writer io.Writer) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	// Gemini 2.5 models enable "thinking" by default, and reasoning tokens
	// count against MaxOutputTokens — on Flash this routinely consumes the
	// majority of a 4096-token budget and truncates the visible Markdown
	// mid-sentence. The narrative analysis is structured output, not a math
	// problem; explicitly zero the thinking budget so the full token budget
	// is spent on the response the user actually reads.
	thinkingBudget := int32(0)
	cfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(SystemPrompt, genai.RoleUser),
		MaxOutputTokens:   g.maxTok,
		ThinkingConfig:    &genai.ThinkingConfig{ThinkingBudget: &thinkingBudget},
	}

	var truncated bool
	for resp, err := range g.client.Models.GenerateContentStream(ctx, g.model, genai.Text(string(body)), cfg) {
		if err != nil {
			return fmt.Errorf("gemini stream: %w", err)
		}
		for _, cand := range resp.Candidates {
			if cand.FinishReason == genai.FinishReasonMaxTokens {
				truncated = true
			}
		}
		text := resp.Text()
		if text == "" {
			continue
		}
		if _, err := io.WriteString(writer, text); err != nil {
			return fmt.Errorf("write streamed token: %w", err)
		}
	}
	if truncated {
		return fmt.Errorf("%w (provider=gemini, max_tokens=%d)", ErrResponseTruncated, g.maxTok)
	}
	return nil
}
