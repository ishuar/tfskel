// Package templates owns the Go text/template files tfskel renders and the
// Renderer that executes them.
//
// Templates are embedded from the files/ tree: tf/ for per-app Terraform
// files, root/ for project-level config written by init, github/ for
// workflow files. NewRendererWithCustomTemplates adds a user directory
// whose *.tmpl files are registered under tf/<name>, replacing an embedded
// template of the same name or adding a new file to the app directory.
//
// Data is the single value every template receives; which fields are set
// depends on the calling command (see docs/templates.md). The function map
// adds string helpers in Go argument order plus StripConstraint and
// BuildMiseTools. The renderer also exposes a hash of each template's text,
// which internal/generate writes into the tfskel-source marker to detect
// template changes on --upgrade.
//
// The package renders strings only. Deciding output paths, injecting
// markers and writing files is internal/generate's job.
package templates
