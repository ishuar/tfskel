package plan

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

// formatCSV outputs analysis as CSV with metadata and resource details using proper CSV escaping
func (f *PlanFormatter) formatCSV(analysis *PlanAnalysis, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write metadata as comments (not CSV escaped)
	comments := []string{
		"# Terraform Plan Analysis",
		"# Terraform Version: " + analysis.TerraformVersion,
		fmt.Sprintf("# Total Resource Changes: %d", analysis.TotalChanges),
		fmt.Sprintf("# Additions: %d, Modifications: %d, Deletions: %d, Replacements: %d",
			analysis.Additions, analysis.Modifications, analysis.Deletions, analysis.Replacements),
		fmt.Sprintf("# Output Changes: %d (added: %d, deleted: %d, modified: %d, replaced: %d)",
			len(analysis.OutputChanges), analysis.OutputAdditions, analysis.OutputDeletions, analysis.OutputModifications, analysis.OutputReplacements),
		"#",
	}
	for _, comment := range comments {
		if _, err := fmt.Fprintln(w, comment); err != nil {
			return fmt.Errorf("failed to write comment: %w", err)
		}
	}

	// Write header using CSV writer for consistency
	if err := csvWriter.Write([]string{"Address", "Type", "Name", "Provider", "Action", "Severity"}); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Sort resources by severity before writing
	sortedResources := sortResourcesBySeverity(analysis.ResourceChanges)

	// Write resource data with proper CSV escaping
	for _, rc := range sortedResources {
		if err := csvWriter.Write([]string{
			rc.Address,
			rc.Type,
			rc.Name,
			rc.Provider,
			rc.ActionString,
			string(rc.Severity), // Convert Severity type to string
		}); err != nil {
			return fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	// Write output changes section if present
	if len(analysis.OutputChanges) > 0 {
		csvWriter.Flush()
		outputComments := []string{
			"#",
			fmt.Sprintf("# Output Changes: %d total (%d added, %d deleted, %d modified, %d replaced)",
				len(analysis.OutputChanges), analysis.OutputAdditions, analysis.OutputDeletions, analysis.OutputModifications, analysis.OutputReplacements),
			"#",
		}
		for _, comment := range outputComments {
			if _, err := fmt.Fprintln(w, comment); err != nil {
				return fmt.Errorf("failed to write output comment: %w", err)
			}
		}

		if err := csvWriter.Write([]string{"Output", "Action", "Sensitive"}); err != nil {
			return fmt.Errorf("failed to write output CSV header: %w", err)
		}
		for _, oc := range analysis.OutputChanges {
			if err := csvWriter.Write([]string{
				oc.Name,
				formatActions(oc.Actions),
				strconv.FormatBool(oc.Sensitive),
			}); err != nil {
				return fmt.Errorf("failed to write output CSV record: %w", err)
			}
		}
	}

	return csvWriter.Error()
}
