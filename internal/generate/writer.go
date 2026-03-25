package generate

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ishuar/tfskel/internal/templates"
)

// determineOutputPath converts template path to output location based on category
// Template paths are like: root/.gitignore.tmpl, tf/backend.tf.tmpl, github/workflow.yaml.tmpl
func (g *Generator) determineOutputPath(tmplPath, appPath string, data *templates.Data) (string, bool) {
	// Normalize path separators
	tmplPath = filepath.ToSlash(tmplPath)
	parts := strings.Split(tmplPath, "/")

	if len(parts) < 2 {
		// Invalid template path format
		return "", false
	}

	category := parts[0]
	fileName := strings.TrimSuffix(parts[len(parts)-1], ".tmpl")

	switch category {
	case "root":
		// Place at project root
		projectRoot := "."
		return filepath.Join(projectRoot, fileName), true
	case "tf":
		// Place in app directory
		return filepath.Join(appPath, fileName), true
	case categoryGithub:
		// Place in .github/workflows/ directory at project root
		projectRoot := "."

		// Only .tmpl files get dynamic naming (e.g. terraform-plan-apply.yaml.tmpl → dev-terraform-plan-apply.yaml).
		// Static files (reusable workflows, lint.yaml, etc.) keep their original names.
		isTemplate := strings.HasSuffix(parts[len(parts)-1], ".tmpl")
		if !isTemplate {
			return filepath.Join(projectRoot, ".github", "workflows", fileName), true
		}

		// Generate dynamic workflow name: e.g. dev-terraform-plan-apply.yaml
		dynamicFileName, err := g.generateWorkflowFileName(fileName, data)
		if err != nil {
			g.log.Errorf("Failed to generate workflow filename: %v", err)
			return "", false
		}
		return filepath.Join(projectRoot, ".github", "workflows", dynamicFileName), true
	default:
		// Unknown category
		return "", false
	}
}

// sourceComment returns the source marker comment for the given template and output path,
// or an empty string if the file does not support markers.
func sourceComment(renderer *templates.Renderer, tmplName, outputPath string) string {
	hash := renderer.GetTemplateHash(tmplName)
	if hash == "" {
		return ""
	}
	return SourceCommentForFile(tmplName, hash, outputPath)
}

// writeTemplateFile creates the directory structure, renders the template with all markers, and writes the result.
// Render failures are logged and skipped (returns nil) since this is used during initial generation
// where a single template failure should not block the entire run.
func (g *Generator) writeTemplateFile(tmplPath, outputPath string, data *templates.Data) error {
	// Ensure parent directory exists
	outputDir := filepath.Dir(outputPath)
	if err := g.fs.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
	}

	if err := g.renderAndWriteFile(tmplPath, outputPath, data); err != nil {
		g.log.Infof("Skipping %s: %v", outputPath, err)
		return nil
	}

	// Log success
	templateSource := g.renderer.GetTemplateSource(tmplPath)
	if templateSource == "" {
		templateSource = tmplPath
	}
	if g.dryRun {
		g.log.Infof("[dry-run] Would create %s from %s", outputPath, templateSource)
	} else {
		g.log.Successf("Created %s from %s", outputPath, templateSource)
	}
	g.tracker.Record(OpCreated, outputPath)

	return nil
}

// renderWithMarkers renders a template and injects source and metadata marker comments.
func (g *Generator) renderWithMarkers(tmplName, outputPath string, data *templates.Data) (string, error) {
	content, err := g.renderer.Render(tmplName, data)
	if err != nil {
		return "", fmt.Errorf("failed to render %s: %w", tmplName, err)
	}

	if comment := sourceComment(g.renderer, tmplName, outputPath); comment != "" {
		content = InjectSourceMarker(content, comment)
	}

	ext := filepath.Ext(outputPath)
	metaComment, tagsHashComment := metadataCommentsForTemplate(tmplName, data, ext)
	if metaComment != "" || tagsHashComment != "" {
		content = InjectMetadataMarkers(content, metaComment, tagsHashComment)
	}

	return content, nil
}

// renderAndWriteFile renders a template with all markers and writes it to outputPath.
func (g *Generator) renderAndWriteFile(tmplName, outputPath string, data *templates.Data) error {
	content, err := g.renderWithMarkers(tmplName, outputPath, data)
	if err != nil {
		return err
	}

	if err := g.fs.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	return nil
}
