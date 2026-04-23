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
	if !g.isUpgradeEligible(tmplPath) {
		g.log.Debugf("Template %s not in upgrade list, skipping", tmplPath)
		return nil
	}

	content, err := g.fs.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("failed to read %s for upgrade check: %w", outputPath, err)
	}

	upgradeVerb, forceVerb := g.upgradeVerbs()

	marker, handled, err := g.resolveMarker(tmplPath, outputPath, data, content, forceVerb)
	if err != nil || handled {
		return err
	}
	if marker.Template != tmplPath {
		g.log.Debugf("%s was generated from %s, not %s, skipping", outputPath, marker.Template, tmplPath)
		return nil
	}

	currentHash := g.renderer.GetTemplateHash(tmplPath)
	rendered, err := g.renderWithMarkers(tmplPath, outputPath, data)
	if err != nil {
		return err
	}
	if rendered == string(content) {
		g.log.Debugf("%s is up to date, skipping", outputPath)
		g.tracker.Record(OpSkipped, outputPath)
		return nil
	}

	if marker.Hash != currentHash {
		g.log.Infof("%s %s (template: %s -> %s)", upgradeVerb, outputPath, marker.Hash, currentHash)
	} else {
		g.log.Infof("%s %s (content drift detected)", upgradeVerb, outputPath)
	}
	if err := g.renderAndWriteFile(tmplPath, outputPath, data); err != nil {
		return err
	}
	if !g.dryRun {
		g.log.Successf("Upgraded %s", outputPath)
	}
	g.tracker.Record(OpUpgraded, outputPath)
	return nil
}

// upgradeVerbs returns the (upgrade, force) log verbs with dry-run prefixes applied.
func (g *Generator) upgradeVerbs() (string, string) {
	if g.dryRun {
		return "[dry-run] Would upgrade", "[dry-run] Would force upgrade"
	}
	return "Upgrading", "Force upgrading"
}

// resolveMarker extracts the source marker from existing file content. When the
// marker is missing or malformed, it applies --force semantics and returns
// handled=true to signal the caller that the file has been fully processed.
func (g *Generator) resolveMarker(tmplPath, outputPath string, data *templates.Data, content []byte, forceVerb string) (*SourceMarker, bool, error) {
	marker, err := ExtractSourceMarker(string(content))
	if err == nil {
		return marker, false, nil
	}
	if errors.Is(err, ErrSourceMarkerNotFound) {
		if g.force {
			g.log.Infof("%s %s (--force, no source marker)", forceVerb, outputPath)
			g.tracker.Record(OpForced, outputPath)
			return nil, true, g.renderAndWriteFile(tmplPath, outputPath, data)
		}
		g.log.Infof("%s has no source marker, skipping upgrade (use --force to override)", outputPath)
		g.tracker.Record(OpSkipped, outputPath)
		return nil, true, nil
	}
	if g.force {
		g.log.Infof("%s %s (--force, invalid source marker: %v)", forceVerb, outputPath, err)
		g.tracker.Record(OpForced, outputPath)
		return nil, true, g.renderAndWriteFile(tmplPath, outputPath, data)
	}
	return nil, true, fmt.Errorf("invalid source marker in %s: %w", outputPath, err)
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
