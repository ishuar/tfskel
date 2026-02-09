package drift

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

var (
	// ErrBinaryFormat indicates the plan file is in binary format
	ErrBinaryFormat = errors.New("plan file is in binary format")
	// ErrMissingFormatVersion indicates the plan file is missing format_version
	ErrMissingFormatVersion = errors.New("invalid plan file: missing format_version")
)

// ParsePlanFile reads and parses a terraform plan JSON file
// The plan file must be in JSON format generated with:
//
//	terraform plan -out=tfplan.binary
//	terraform show -json tfplan.binary > tfplan.json
func ParsePlanFile(filename string) (*TerraformPlan, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan file: %w", err)
	}

	// Check if it's likely a binary file (Terraform binary plan format)
	if len(data) > 0 && data[0] != '{' {
		return nil, fmt.Errorf("%w. Convert to JSON with: terraform show -json %s > plan.json", ErrBinaryFormat, filename)
	}

	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("invalid plan file format: %w. Ensure file is valid Terraform plan JSON", err)
	}

	// Validate it's a terraform plan (should have format_version)
	if plan.FormatVersion == "" {
		return nil, fmt.Errorf("%w. This may not be a Terraform plan JSON file", ErrMissingFormatVersion)
	}

	return &plan, nil
}
