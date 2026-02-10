package drift

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

const (
	// headerBufferSize is the size of the buffer used to detect binary files
	headerBufferSize = 128
)

var (
	// ErrBinaryFormat indicates the plan file is in binary format
	ErrBinaryFormat = errors.New("plan file is in binary format")
	// ErrMissingFormatVersion indicates the plan file is missing format_version
	ErrMissingFormatVersion = errors.New("invalid plan file: missing format_version")
)

// ParsePlanFile reads and parses a terraform plan JSON file using streaming decoder
// to reduce memory usage. The plan file must be in JSON format generated with:
//
//	terraform plan -out=tfplan.binary
//	terraform show -json tfplan.binary > tfplan.json
func ParsePlanFile(filename string) (*TerraformPlan, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open plan file: %w", err)
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("failed to close plan file: %w", closeErr)
		}
	}()

	// Check if file appears to be binary format
	if err := checkBinaryFormat(file, filename); err != nil {
		return nil, err
	}

	// Reset file position to start for JSON decoding
	if _, err := file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file position: %w", err)
	}

	// Use streaming decoder to reduce memory usage
	var plan TerraformPlan
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&plan); err != nil {
		return nil, fmt.Errorf("invalid plan file format: %w. Ensure file is valid Terraform plan JSON", err)
	}

	// Validate it's a terraform plan (should have format_version)
	if plan.FormatVersion == "" {
		return nil, fmt.Errorf("%w. This may not be a Terraform plan JSON file", ErrMissingFormatVersion)
	}

	return &plan, nil
}

// checkBinaryFormat reads the first few bytes to detect if file is in binary format.
// This function provides better error messages for a common user mistake: passing a
// binary Terraform plan file instead of the required JSON format.
//
// Common scenario:
//   - User runs: terraform plan -out=tfplan.binary
//   - User forgets: terraform show -json tfplan.binary > tfplan.json
//   - User tries: tfskel drift plan --plan-file tfplan.binary
//   - Without this check: cryptic JSON parse error
//   - With this check: clear error with fix command
//
// The function:
//  1. Reads first 128 bytes (headerBufferSize) to peek at file content
//  2. Handles UTF-8 BOM (0xEF, 0xBB, 0xBF) that some editors add
//  3. Skips leading whitespace (valid JSON can start with spaces/newlines)
//  4. Checks if first non-whitespace character is '{' (start of JSON object)
//  5. Returns helpful error with remediation if not JSON
//
// Note: We check len(header) >= 3 in addition to n >= 3 to satisfy gosec linter
// which requires explicit bounds checking to prevent potential slice index panics.
func checkBinaryFormat(file *os.File, filename string) error {
	header := make([]byte, headerBufferSize)
	n, err := file.Read(header)
	if err != nil && n == 0 {
		return fmt.Errorf("failed to read plan file: %w", err)
	}

	if n == 0 {
		return nil // Empty file, let JSON decoder handle it
	}

	// Skip UTF-8 BOM if present (0xEF, 0xBB, 0xBF)
	offset := 0
	if n >= 3 && len(header) >= 3 && header[0] == 0xEF && header[1] == 0xBB && header[2] == 0xBF {
		offset = 3
	}

	// Find first non-whitespace byte
	for i := offset; i < n; i++ {
		b := header[i]
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			continue
		}
		// If first non-whitespace byte is not '{', it's likely a binary plan file
		if b != '{' {
			return fmt.Errorf("%w. Convert to JSON with: terraform show -json %s > plan.json", ErrBinaryFormat, filename)
		}
		break
	}

	return nil
}
