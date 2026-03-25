package generate

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/ishuar/tfskel/internal/templates"
)

// upgradeFileIfEligible checks whether an existing file should be upgraded from its template.
// It reads the file's source marker, re-renders the template, and compares the full rendered
// content against the existing file. This detects both template source changes and manual edits
// to the rendered file (content drift).
// Returns nil if the file was skipped or upgraded successfully.
func (g *Generator) upgradeFileIfEligible(tmplPath, outputPath string, data *templates.Data) error {
	// Check upgrade whitelist
	if !g.isUpgradeEligible(tmplPath) {
		g.log.Debugf("Template %s not in upgrade list, skipping", tmplPath)
		return nil
	}

	// Read existing file and extract source marker
	content, err := g.fs.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s for upgrade check: %w", outputPath, err)
	}

	upgradeVerb := "Upgrading"
	forceVerb := "Force upgrading"
	if g.dryRun {
		upgradeVerb = "[dry-run] Would upgrade"
		forceVerb = "[dry-run] Would force upgrade"
	}

	marker, err := ExtractSourceMarker(string(content))
	if err != nil {
		if errors.Is(err, ErrSourceMarkerNotFound) {
			if g.force {
				g.log.Infof("%s %s (--force, no source marker)", forceVerb, outputPath)
				g.tracker.Record(OpForced, outputPath)
				return g.renderAndWriteFile(tmplPath, outputPath, data)
			}
			g.log.Infof("%s has no source marker, skipping upgrade (use --force to override)", outputPath)
			g.tracker.Record(OpSkipped, outputPath)
			return nil
		}
		// Malformed source marker (e.g. invalid JSON)
		if g.force {
			g.log.Infof("%s %s (--force, invalid source marker: %v)", forceVerb, outputPath, err)
			g.tracker.Record(OpForced, outputPath)
			return g.renderAndWriteFile(tmplPath, outputPath, data)
		}
		return fmt.Errorf("invalid source marker in %s: %w", outputPath, err)
	}

	// Verify the marker template matches this template
	if marker.Template != tmplPath {
		g.log.Debugf("%s was generated from %s, not %s, skipping", outputPath, marker.Template, tmplPath)
		return nil
	}

	// Compare template hash to detect template source changes vs content drift
	currentHash := g.renderer.GetTemplateHash(tmplPath)
	templateChanged := marker.Hash != currentHash

	// Re-render the template with markers and compare to detect any drift
	rendered, err := g.renderWithMarkers(tmplPath, outputPath, data)
	if err != nil {
		return err
	}

	if rendered == string(content) {
		g.log.Debugf("%s is up to date, skipping", outputPath)
		g.tracker.Record(OpSkipped, outputPath)
		return nil
	}

	// Content differs — provide specific reason
	if templateChanged {
		g.log.Infof("%s %s (template: %s -> %s)", upgradeVerb, outputPath, marker.Hash, currentHash)
	} else {
		g.log.Infof("%s %s (content drift detected)", upgradeVerb, outputPath)
	}
	if err := g.renderAndWriteFile(tmplPath, outputPath, data); err != nil {
		return err
	}
	if g.dryRun {
		g.log.Infof("[dry-run] Would upgrade %s", outputPath)
	} else {
		g.log.Successf("Upgraded %s", outputPath)
	}
	g.tracker.Record(OpUpgraded, outputPath)
	return nil
}

// isUpgradeEligible checks if a template is eligible for upgrade based on the config whitelist.
// If no whitelist is configured, all templates are eligible.
func (g *Generator) isUpgradeEligible(tmplPath string) bool {
	if g.config.Templates == nil || len(g.config.Templates.Upgrade) == 0 {
		return true // no whitelist = all eligible
	}
	baseName := filepath.Base(tmplPath)
	return slices.Contains(g.config.Templates.Upgrade, baseName)
}
