package review

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ishuar/tfskel/internal/ai"
	"github.com/ishuar/tfskel/internal/format"
	"github.com/ishuar/tfskel/internal/plan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_UnsupportedFormatFails(t *testing.T) {
	req := baseRequest(t, planWithChanges)
	req.Format = format.OutputFormat("yaml")

	var out bytes.Buffer
	_, err := Run(context.Background(), req, &out, testLogger())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to format output")
}

func TestRun_JSONFormat(t *testing.T) {
	req := baseRequest(t, planWithChanges)
	req.Format = format.FormatJSON

	var out bytes.Buffer
	result, err := Run(context.Background(), req, &out, testLogger())

	require.NoError(t, err)
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode)
	assert.Contains(t, out.String(), `"total_changes"`)
}

// brokenAfterWriter succeeds for the first n writes, then fails — simulating
// a pipe that breaks between the report and the AI section.
type brokenAfterWriter struct {
	n int
}

func (w *brokenAfterWriter) Write(p []byte) (int, error) {
	if w.n <= 0 {
		return 0, errors.New("pipe closed")
	}
	w.n--
	return len(p), nil
}

// TestRun_AIHeaderWriteFailureIsAdditive covers the writer-failure branch in
// the AI append: the report succeeds, the AI section header write fails, and
// the run still completes with the analysis exit code.
func TestRun_AIHeaderWriteFailureIsAdditive(t *testing.T) {
	stub := &stubAIClient{text: "never reached"}
	req := baseRequest(t, planWithChanges)
	req.Format = format.FormatJSON // exactly one report write, so write #2 is the AI header
	req.NewAIClient = func(context.Context) (ai.Client, error) { return stub, nil }

	result, err := Run(context.Background(), req, &brokenAfterWriter{n: 1}, testLogger())

	require.NoError(t, err, "a broken pipe during the AI append must stay additive")
	assert.Equal(t, plan.ExitCodeCritical, result.ExitCode)
	assert.Nil(t, stub.payload, "Explain must not run when the header write fails")
}
