package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ishuar/tfskel/internal/templates"
)

// sanitizeWorkflowFileName validates and sanitizes a workflow filename to prevent path traversal
// Returns the sanitized filename and a boolean indicating if it's valid
func sanitizeWorkflowFileName(filename string) (string, bool) {
	// Reject paths containing dangerous characters or patterns BEFORE cleaning
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		return "", false
	}

	// Reject empty or just extension
	if filename == "" || filename == ".yaml" || filename == "." {
		return "", false
	}

	// Additional validation: ensure the cleaned version matches the input
	cleaned := filepath.Clean(filename)
	if cleaned != filename {
		return "", false
	}

	return filename, true
}

// generateWorkflowFileName creates dynamic workflow file names based on template data.
// Default pattern: {env}-terraform-plan-apply.yaml (e.g., dev-terraform-plan-apply.yaml)
// If name is provided in config, it uses that as a plain string (no Go template rendering).
// Pattern with name: {env}-{name}.yaml (e.g., dev-my-terraform.yaml)
// Returns error if name contains Go template syntax or produces invalid filename.
func (g *Generator) generateWorkflowFileName(originalFileName string, data *templates.Data) (string, error) {
	// Default path: no custom name configured
	if g.config.Workflows == nil ||
		g.config.Workflows.Name == "" {
		return g.generateDefaultWorkflowFileName(originalFileName, data), nil
	}

	name := g.config.Workflows.Name

	// Reject Go template syntax — name must be a plain string
	if strings.Contains(name, "{{") || strings.Contains(name, "}}") {
		return "", fmt.Errorf("%w: name must be a plain string without Go template syntax (e.g. 'my-terraform'), got: %s", ErrInvalidWorkflowFileName, name)
	}

	// Normalize: strip trailing .yaml if user included it
	name, _ = strings.CutSuffix(name, ".yaml")

	// Build filename: {env}-{name}.yaml
	baseFileName := data.Env + "-" + name + ".yaml"

	// Validate and sanitize the filename to prevent path traversal
	sanitized, valid := sanitizeWorkflowFileName(baseFileName)
	if !valid {
		return "", fmt.Errorf("%w (potential path traversal): %s", ErrInvalidWorkflowFileName, baseFileName)
	}

	return sanitized, nil
}

// generateDefaultWorkflowFileName creates the default workflow file name
// Pattern: {env}-{workflowType}.yaml (e.g., dev-terraform-plan-apply.yaml, dev-terraform-destroy.yaml)
// The workflowType is derived from the template filename by stripping .yaml
// (e.g., "terraform-plan-apply.yaml" -> "terraform-plan-apply")
func (g *Generator) generateDefaultWorkflowFileName(originalFileName string, data *templates.Data) string {
	// Extract the workflow type from the original filename
	// e.g., "terraform-plan-apply.yaml" -> "terraform-plan-apply"
	workflowType := strings.TrimSuffix(originalFileName, ".yaml")

	return fmt.Sprintf("%s-%s.yaml", data.Env, workflowType)
}
