// Package bootstrap creates a new tfskel project from zero: root config files,
// environment/region skeleton, optional GitHub workflow files, and the default
// .tfskel.yaml. It backs the `tfskel init` command.
//
// Paired mental model with internal/generate:
//   - bootstrap: project-level, one-time, from-zero scaffolding
//   - generate:  per-app, repeatable file generation under envs/<env>/<region>/<app>/
package bootstrap

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/generate"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
)

// defaultTerraformVersion is the Terraform version used when no .tfskel.yaml is present.
const defaultTerraformVersion = "1.13.1"

// initManagedFile pairs a target filename with its source template.
type initManagedFile struct {
	filename     string
	templateName string
}

// rootConfigFiles lists root config files managed by init (rendered with nil data).
var rootConfigFiles = []initManagedFile{
	{".pre-commit-config.yaml", "root/.pre-commit-config.yaml.tmpl"},
	{".tflint.hcl", "root/.tflint.hcl.tmpl"},
	{"trivy.yaml", "root/trivy.yaml.tmpl"},
}

// staticWorkflowFiles lists the reusable GitHub workflow files created when workflows are enabled.
var staticWorkflowFiles = []initManagedFile{
	{"lint.yaml", "github/lint.yaml"},
	{"reusable-detect-changes.yaml", "github/reusable-detect-changes.yaml"},
	{"reusable-terraform-plan-apply.yaml", "github/reusable-terraform-plan-apply.yaml"},
	{"reusable-lint.yaml", "github/reusable-lint.yaml"},
}

// Runner bundles dependencies for init command file operations.
type Runner struct {
	fs       fs.FileSystem
	log      *logger.Logger
	renderer *templates.Renderer
	upgrade  bool
	force    bool
	dryRun   bool
	skip     map[string]bool
}

// Options configures a Runner's behavior. Zero-value Options creates a plain
// first-time-bootstrap runner: no upgrade, no force, not a dry run, no skip list.
type Options struct {
	// Upgrade re-renders init-managed files whose source template has changed.
	Upgrade bool
	// Force, combined with Upgrade, overwrites files that lack a source marker.
	Force bool
	// DryRun logs what would happen without writing to disk (pair with a DryRunFileSystem).
	DryRun bool
	// Skip lists basenames to skip during upgrade.
	Skip map[string]bool
}

// NewRunner constructs a Runner and initializes the template renderer.
func NewRunner(filesystem fs.FileSystem, log *logger.Logger, opts Options) (*Runner, error) {
	renderer, err := templates.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create renderer: %w", err)
	}
	return &Runner{
		fs:       filesystem,
		log:      log,
		renderer: renderer,
		upgrade:  opts.Upgrade,
		force:    opts.Force,
		dryRun:   opts.DryRun,
		skip:     opts.Skip,
	}, nil
}

// CreateProjectStructure materializes the full project skeleton: base directory,
// root config files, user-owned seed files (.gitignore, .mise.toml), .tfskel.yaml,
// envs/<env>/<region>/ tree, and optional shared GitHub workflow files.
func (r *Runner) CreateProjectStructure(baseDir, terraformVersion string, regions, environments []string, createWorkflows bool) error {
	if err := r.fs.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	for _, file := range rootConfigFiles {
		if err := r.createFileFromTemplate(filepath.Join(baseDir, file.filename), file.templateName, nil); err != nil {
			return err
		}
	}

	// .gitignore and .mise.toml are user-owned after initial creation:
	// tfskel seeds them once, without a source marker, and never touches them on --upgrade.
	if err := r.createUnmanagedFile(filepath.Join(baseDir, ".gitignore"), "root/.gitignore.tmpl", nil); err != nil {
		return err
	}
	if err := r.createUnmanagedFile(filepath.Join(baseDir, ".mise.toml"), "root/.mise.toml.tmpl", &templates.Data{
		TerraformVersion: terraformVersion,
		Environments:     environments,
	}); err != nil {
		return err
	}

	if err := r.CreateDefaultConfig(filepath.Join(baseDir, ".tfskel.yaml")); err != nil {
		return err
	}

	r.log.Debugf("Creating directory structure for %d environment(s): %v", len(environments), environments)
	for _, env := range environments {
		envPath := filepath.Join(baseDir, "envs", env)

		tfVersionPath := filepath.Join(envPath, ".terraform-version")
		tfVersionData := &templates.Data{TerraformVersion: terraformVersion}
		if err := r.createFileFromTemplate(tfVersionPath, "root/.terraform-version.tmpl", tfVersionData); err != nil {
			return err
		}

		for _, region := range regions {
			regionPath := filepath.Join(envPath, region)
			relPath, relErr := filepath.Rel(baseDir, regionPath)
			if relErr != nil {
				relPath = regionPath
			}

			if r.fs.DirExists(regionPath) {
				r.log.Infof("Directory %s/ already exists", relPath)
				continue
			}

			if err := r.fs.MkdirAll(regionPath, 0755); err != nil {
				return fmt.Errorf("failed to create region directory %s: %w", regionPath, err)
			}

			if r.dryRun {
				r.log.Infof("[dry-run] Would create directory: %s/", relPath)
			} else {
				r.log.Successf("Created directory: %s/", relPath)
			}
		}
	}

	if createWorkflows {
		for _, file := range staticWorkflowFiles {
			targetPath := filepath.Join(baseDir, ".github", "workflows", file.filename)
			if err := r.createFileFromTemplate(targetPath, file.templateName, nil); err != nil {
				return err
			}
		}
	}

	return nil
}

// createFileFromTemplate renders templateName and writes it to targetPath. Existing
// files are skipped unless upgrade mode is on, in which case upgradeFile handles
// the source-marker comparison and potential re-render.
func (r *Runner) createFileFromTemplate(targetPath, templateName string, data *templates.Data) error {
	logPath := r.relLogPath(targetPath)

	if r.fs.FileExists(targetPath) {
		if r.upgrade {
			if r.skip[filepath.Base(targetPath)] {
				r.log.Infof("%s skipped (--skip)", logPath)
				return nil
			}
			return r.upgradeFile(targetPath, templateName, data, logPath)
		}
		r.log.Infof("%s already exists, skipping", logPath)
		return nil
	}

	if r.skip[filepath.Base(targetPath)] {
		r.log.Debugf("%s in --skip list but does not exist yet, creating normally", logPath)
	}

	content, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}

	if comment := generate.SourceCommentForFile(templateName, r.renderer.GetTemplateHash(templateName), targetPath); comment != "" {
		content = generate.InjectSourceMarker(content, comment)
	}

	if err := r.writeFile(targetPath, content); err != nil {
		return err
	}

	if r.dryRun {
		r.log.Infof("[dry-run] Would create %s", logPath)
	} else {
		r.log.Successf("Created %s", logPath)
	}
	return nil
}

// createUnmanagedFile renders a template and writes it only if the file does not
// already exist. It does not inject a source marker and is always skipped during
// --upgrade, because the file is user-owned after initial creation.
func (r *Runner) createUnmanagedFile(targetPath, templateName string, data *templates.Data) error {
	logPath := r.relLogPath(targetPath)

	if r.fs.FileExists(targetPath) {
		r.log.Debugf("%s already exists, skipping (user-owned)", logPath)
		return nil
	}

	content, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}

	if err := r.writeFile(targetPath, content); err != nil {
		return err
	}

	if r.dryRun {
		r.log.Infof("[dry-run] Would create %s", logPath)
	} else {
		r.log.Successf("Created %s", logPath)
	}
	return nil
}

// upgradeFile handles the upgrade logic for an existing init-managed file.
func (r *Runner) upgradeFile(targetPath, templateName string, data *templates.Data, logPath string) error {
	existingContent, err := r.fs.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("failed to read %s for upgrade check: %w", logPath, err)
	}

	upgradeVerb := "Upgrading"
	forceVerb := "Force upgrading"
	if r.dryRun {
		upgradeVerb = "[dry-run] Would upgrade"
		forceVerb = "[dry-run] Would force upgrade"
	}

	marker, markerErr := generate.ExtractSourceMarker(string(existingContent))

	switch {
	case errors.Is(markerErr, generate.ErrSourceMarkerNotFound) && !r.force:
		r.log.Infof("%s has no source marker, skipping upgrade (use --force to override)", logPath)
		return nil
	case errors.Is(markerErr, generate.ErrSourceMarkerNotFound):
		r.log.Infof("%s %s (--force, no source marker)", forceVerb, logPath)
	case markerErr != nil && !r.force:
		return fmt.Errorf("invalid source marker in %s: %w", logPath, markerErr)
	case markerErr != nil:
		r.log.Infof("%s %s (--force, invalid source marker: %v)", forceVerb, logPath, markerErr)
	}

	if markerErr == nil {
		return r.upgradeFromMarker(targetPath, templateName, data, logPath, marker, existingContent, upgradeVerb)
	}

	// No valid marker (--force path): re-render and overwrite.
	content, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}
	if comment := generate.SourceCommentForFile(templateName, r.renderer.GetTemplateHash(templateName), targetPath); comment != "" {
		content = generate.InjectSourceMarker(content, comment)
	}

	return r.writeFile(targetPath, content)
}

// upgradeFromMarker handles upgrade when a valid source marker is present.
// It compares re-rendered content against the existing file to detect both
// template changes and data drift (e.g. version bumps in .tfskel.yaml).
func (r *Runner) upgradeFromMarker(targetPath, templateName string, data *templates.Data, logPath string, marker *generate.SourceMarker, existingContent []byte, upgradeVerb string) error {
	if marker.Template != templateName {
		r.log.Debugf("%s source marker template mismatch (%s != %s), skipping", logPath, marker.Template, templateName)
		return nil
	}

	rendered, err := r.renderTemplate(templateName, data)
	if err != nil {
		return err
	}
	if comment := generate.SourceCommentForFile(templateName, r.renderer.GetTemplateHash(templateName), targetPath); comment != "" {
		rendered = generate.InjectSourceMarker(rendered, comment)
	}

	if rendered == string(existingContent) {
		r.log.Debugf("%s is up to date, skipping", logPath)
		return nil
	}

	currentHash := r.renderer.GetTemplateHash(templateName)
	if marker.Hash != currentHash {
		r.log.Infof("%s %s (template: %s -> %s)", upgradeVerb, logPath, marker.Hash, currentHash)
	} else {
		r.log.Infof("%s %s (config drift detected)", upgradeVerb, logPath)
	}

	return r.writeFile(targetPath, rendered)
}

// renderTemplate renders templateName with the given data, normalizing nil to an empty Data value.
// Nil is a legitimate call convention for templates that don't need any data.
func (r *Runner) renderTemplate(templateName string, data *templates.Data) (string, error) {
	if data == nil {
		data = &templates.Data{}
	}
	return r.renderer.Render(templateName, data)
}

// writeFile writes content to targetPath via the filesystem abstraction.
// In dry-run mode the DryRunFileSystem silently skips the actual write.
// Callers log the action with the appropriate verb.
func (r *Runner) writeFile(targetPath, content string) error {
	if err := r.fs.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", targetPath, err)
	}
	return nil
}

// relLogPath returns targetPath relative to the current working directory for
// user-friendly log output, falling back to targetPath if cwd cannot be resolved.
func (r *Runner) relLogPath(targetPath string) string {
	if cwd, err := os.Getwd(); err == nil {
		if rel, err := filepath.Rel(cwd, targetPath); err == nil {
			return rel
		}
	}
	return targetPath
}
