package ai

import (
	"context"
	"errors"
	"io"
)

// ErrResponseTruncated is returned by Explain implementations when the model
// hit the output-token ceiling before completing its response. The streamed
// text up to the cut-off has already been written to the caller's writer —
// the error exists so the caller can surface a clear warning telling the user
// to raise --ai-max-tokens (or the ai.max_tokens config key) rather than
// silently shipping a half-finished analysis.
var ErrResponseTruncated = errors.New("AI response truncated at max_tokens limit")

// Client streams a Markdown narrative analysis of a plan Payload to writer.
//
// Implementations return the underlying API error directly; the caller is
// responsible for treating a non-nil error as additive failure (warn, do not
// propagate) so that AI unavailability never breaks the `review plan` exit code.
type Client interface {
	Explain(ctx context.Context, payload *Payload, writer io.Writer) error
	// Provider returns the backend identifier (e.g. "anthropic", "gemini").
	Provider() string
	// Model returns the resolved model name used for requests.
	Model() string
}
