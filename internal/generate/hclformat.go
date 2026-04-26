package generate

import (
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

// hclExtensions are the file extensions reformatted by `terraform fmt` /
// hclwrite.Format. Fixed by the HCL ecosystem — no dynamic source exists
// (terraform itself hardcodes this list).
var hclExtensions = map[string]struct{}{
	".tf":     {},
	".tfvars": {},
	".hcl":    {},
}

// isHCLOutput reports whether the output path's extension is one that
// `terraform fmt` (and therefore hclwrite.Format) reformats.
func isHCLOutput(outputPath string) bool {
	_, ok := hclExtensions[strings.ToLower(filepath.Ext(outputPath))]
	return ok
}

// formatIfHCL runs hclwrite.Format on the content when outputPath is an HCL
// file, so the bytes we write match what `terraform fmt` would produce.
// Without this, every `--upgrade` after a `terraform fmt` run shows false
// "content drift" because the template emits unaligned `=` while tf fmt
// column-aligns them.
func formatIfHCL(outputPath, content string) string {
	if !isHCLOutput(outputPath) {
		return content
	}
	return string(hclwrite.Format([]byte(content)))
}
