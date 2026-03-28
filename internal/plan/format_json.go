package plan

import (
	"encoding/json"
	"io"
)

// jsonFilterMeta holds filter metadata for JSON output.
type jsonFilterMeta struct {
	ActiveFilters   []string `json:"active_filters"`
	TotalUnfiltered int      `json:"total_unfiltered"`
}

// jsonOutput wraps PlanAnalysis with optional filter metadata.
type jsonOutput struct {
	*PlanAnalysis

	Filter *jsonFilterMeta `json:"filter,omitempty"`
}

// formatJSON outputs analysis as JSON
func (f *PlanFormatter) formatJSON(analysis *PlanAnalysis, w io.Writer) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if len(f.activeFilters) > 0 {
		return encoder.Encode(jsonOutput{
			PlanAnalysis: analysis,
			Filter: &jsonFilterMeta{
				ActiveFilters:   f.activeFilters,
				TotalUnfiltered: f.totalResourceCount,
			},
		})
	}

	return encoder.Encode(analysis)
}
