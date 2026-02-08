package drift

import (
	"encoding/json"
	"fmt"
	"os"
)

// ParsePlanFile reads and parses a terraform plan JSON file
// The plan file must be in JSON format generated with:
//   terraform plan -out=tfplan.binary
//   terraform show -json tfplan.binary > tfplan.json
func ParsePlanFile(filename string) (*TerraformPlan, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	// Check if it's likely a binary file (Terraform binary plan format)
	if len(data) > 0 && data[0] != '{' {
		return nil, fmt.Errorf("plan file appears to be in binary format. Convert to JSON with: terraform show -json %s > plan.json", filename)
	}

	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("invalid plan file format: %w. Ensure file is valid Terraform plan JSON", err)
	}

	// Validate it's a terraform plan (should have format_version)
	if plan.FormatVersion == "" {
		return nil, fmt.Errorf("invalid plan file: missing format_version. This may not be a Terraform plan JSON file")
	}

	return &plan, nil
}
