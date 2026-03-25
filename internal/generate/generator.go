package generate

import (
	"errors"
	"fmt"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
)

// Template category and name constants
const (
	categoryGithub = "github"
	tmplBackendTF  = "tf/backend.tf.tmpl"
	tmplVersionsTF = "tf/versions.tf.tmpl"
)

var (
	// ErrMetadataKeyNotFound indicates the requested metadata key was not found in template metadata
	ErrMetadataKeyNotFound = errors.New("metadata key not found")

	// ErrInvalidWorkflowFileName indicates the name_template produced an invalid filename
	ErrInvalidWorkflowFileName = errors.New("name_template produced invalid filename")
)

// Generator orchestrates the Terraform project generation
//
// Thread-safety: Generator is designed for single-threaded CLI usage.
// The config field is read-only after initialization via NewGenerator.
// The renderer field is set once during Run() and then only read.
// Concurrent use of the same Generator instance is not supported.
type Generator struct {
	config   *config.Config
	fs       fs.FileSystem
	log      *logger.Logger
	renderer *templates.Renderer
	upgrade  bool // When true, re-render files whose template has changed
	force    bool // When true (with upgrade), overwrite files even without source markers
}

// NewGenerator creates a new Generator instance
func NewGenerator(cfg *config.Config, filesystem fs.FileSystem, log *logger.Logger) *Generator {
	return &Generator{
		config: cfg,
		fs:     filesystem,
		log:    log,
	}
}

// SetUpgrade enables the upgrade mode, optionally with force.
// When upgrade is true, files whose source template has changed are re-rendered.
// When force is also true, files without source markers are overwritten unconditionally.
func (g *Generator) SetUpgrade(upgrade, force bool) {
	g.upgrade = upgrade
	g.force = force
}

// initRenderer initializes the template renderer, using custom templates if configured.
func (g *Generator) initRenderer() error {
	var (
		renderer *templates.Renderer
		err      error
	)
	// Defensive: check Templates is initialized (always true from config.Load, but defensive for direct usage)
	if g.config.Templates != nil && g.config.Templates.Dir != "" {
		g.log.Infof("Using custom templates from: %s", g.config.Templates.Dir)
		renderer, err = templates.NewRendererWithCustomTemplates(g.config.Templates.Dir)
	} else {
		g.log.Debug("Using default embedded templates")
		renderer, err = templates.NewRenderer()
	}
	if err != nil {
		return fmt.Errorf("failed to initialize template renderer: %w", err)
	}
	g.renderer = renderer
	return nil
}
