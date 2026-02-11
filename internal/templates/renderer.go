package templates

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	// ErrCustomTemplateDirNotExist indicates the specified custom template directory does not exist
	ErrCustomTemplateDirNotExist = errors.New("custom template directory does not exist")
	// ErrTemplateNotFound indicates the requested template was not found
	ErrTemplateNotFound = errors.New("template not found")
)

// stripConstraint removes version constraint operators (~>, >=, <=, >, <, =) and returns just the version number
// Example: "~> 1.14.3" -> "1.14.3"
func stripConstraint(version string) string {
	// Remove common constraint operators
	version = strings.ReplaceAll(version, "~>", "")
	version = strings.ReplaceAll(version, ">=", "")
	version = strings.ReplaceAll(version, "<=", "")
	version = strings.ReplaceAll(version, ">", "")
	version = strings.ReplaceAll(version, "<", "")
	version = strings.ReplaceAll(version, "=", "")
	return strings.TrimSpace(version)
}

// funcMap provides common template functions for both default and custom templates
var funcMap = template.FuncMap{
	"replace":         strings.ReplaceAll,
	"toLower":         strings.ToLower,
	"toUpper":         strings.ToUpper,
	"trimSpace":       strings.TrimSpace,
	"trimPrefix":      strings.TrimPrefix,
	"trimSuffix":      strings.TrimSuffix,
	"hasPrefix":       strings.HasPrefix,
	"hasSuffix":       strings.HasSuffix,
	"contains":        strings.Contains,
	"join":            strings.Join,
	"split":           strings.Split,
	"stripConstraint": stripConstraint,
}

//go:embed files/**/*.tmpl
var embeddedTemplates embed.FS

// templateFS is the sub-filesystem without the "files/" prefix
var defaultTemplateFS fs.FS

func init() {
	var err error
	defaultTemplateFS, err = fs.Sub(embeddedTemplates, "files")
	if err != nil {
		panic(fmt.Sprintf("failed to create template sub-filesystem: %v", err))
	}
}

// Data holds all the data needed for template rendering
type Data struct {
	Env                string
	Region             string
	AppDir             string
	AccountID          string
	ShortRegion        string
	S3BucketName       string
	TerraformVersion   string
	AWSProviderVersion string
	DefaultTags        map[string]string
}

// Renderer handles template rendering
type Renderer struct {
	templates map[string]*template.Template
	sources   map[string]string // Track where each template came from (for logging)
}

// NewRenderer creates a new template renderer with default embedded templates
func NewRenderer() (*Renderer, error) {
	return NewRendererWithCustomTemplates("", []string{"tf.tmpl"})
}

// NewRendererWithCustomTemplates creates a renderer that supports custom template directory
// allowedExtensions specifies which file extensions to load from custom directory (e.g., ["tf.tmpl", "md.tmpl"])
func NewRendererWithCustomTemplates(customTemplateDir string, allowedExtensions []string) (*Renderer, error) {
	r := &Renderer{
		templates: make(map[string]*template.Template),
		sources:   make(map[string]string),
	}

	// Load default embedded templates
	if err := r.loadTemplatesFromFS(defaultTemplateFS); err != nil {
		return nil, err
	}

	// Load custom templates if directory provided
	if customTemplateDir != "" {
		if err := r.loadCustomTemplates(customTemplateDir, allowedExtensions); err != nil {
			return nil, fmt.Errorf("failed to load custom templates: %w", err)
		}
	}

	return r, nil
}

// loadTemplatesFromFS loads templates from a filesystem
func (r *Renderer) loadTemplatesFromFS(fsys fs.FS) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".tmpl") {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", path, err)
		}

		tmpl, err := template.New(path).Funcs(funcMap).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		r.templates[path] = tmpl
		r.sources[path] = "embedded:" + path
		return nil
	})
}

// loadCustomTemplates loads templates from a custom directory
// Only processes files with extensions in allowedExtensions list
// Example with ["tf.tmpl", "md.tmpl"]:
//
//	backend.tf.tmpl -> tf/backend.tf.tmpl (overrides default)
//	readme.md.tmpl  -> tf/readme.md.tmpl (new template)
func (r *Renderer) loadCustomTemplates(customDir string, allowedExtensions []string) error {
	if _, err := os.Stat(customDir); os.IsNotExist(err) {
		return fmt.Errorf("%w: %s", ErrCustomTemplateDirNotExist, customDir)
	}

	return filepath.Walk(customDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		filename := filepath.Base(path)

		// Check if file matches any allowed extension
		allowed := false
		for _, ext := range allowedExtensions {
			if strings.HasSuffix(filename, "."+ext) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read custom template %s: %w", path, err)
		}

		// Map to tf/ directory
		templateKey := filepath.Join("tf", filename)

		tmpl, err := template.New(templateKey).Funcs(funcMap).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse custom template %s: %w", filename, err)
		}

		r.templates[templateKey] = tmpl
		r.sources[templateKey] = path
		return nil
	})
}

// GetTemplateNames returns all loaded template names
func (r *Renderer) GetTemplateNames() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// Render renders a template with the provided data
func (r *Renderer) Render(templateName string, data *Data) (string, error) {
	tmpl, ok := r.templates[templateName]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrTemplateNotFound, templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.String(), nil
}

// GetTemplateSource returns the source path of a template (for logging purposes)
// Returns empty string if template not found
func (r *Renderer) GetTemplateSource(templateName string) string {
	if source, ok := r.sources[templateName]; ok {
		return source
	}
	return ""
}
