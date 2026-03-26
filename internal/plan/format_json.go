package plan

import (
	"encoding/json"
	"io"
)

// formatJSON outputs analysis as JSON
func (f *PlanFormatter) formatJSON(analysis *PlanAnalysis, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(analysis)
}
