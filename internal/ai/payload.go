package ai

import (
	"encoding/json"

	"github.com/ishuar/tfskel/internal/plan"
)

// redactedMarker replaces any attribute value Terraform marked sensitive.
// Constant string — readers grepping for "<redacted>" in logs or test output
// should see one canonical form.
const redactedMarker = "<redacted>"

// maxAttrValueChars caps a single attribute value before transmission. A
// runaway user_data blob or rendered template should not blow the token
// budget on its own. Truncation is marked with truncationSuffix so the model
// knows the value was cut.
const (
	maxAttrValueChars = 500
	truncationSuffix  = "...<truncated>"
)

// Payload is the wire-format sent to the model as the user message. It mirrors
// the structure of `terraform show -json` output but strips sensitive values,
// drops unchanged attributes, and caps oversized strings.
type Payload struct {
	TerraformVersion  string            `json:"terraform_version"`
	Counts            Counts            `json:"counts"`
	CriticalResources []string          `json:"critical_resources,omitempty"`
	Resources         []ResourcePayload `json:"resources"`
	OutputChanges     []OutputPayload   `json:"output_changes,omitempty"`
}

// Counts mirrors the aggregate change counts displayed in the table so the
// model has the same totals the engineer already saw.
type Counts struct {
	Total         int `json:"total"`
	Additions     int `json:"additions"`
	Modifications int `json:"modifications"`
	Deletions     int `json:"deletions"`
	Replacements  int `json:"replacements"`
}

// ResourcePayload describes a single resource change. Before/After maps contain
// only attributes that actually changed; sensitive attributes are replaced
// with redactedMarker; oversized values are truncated.
type ResourcePayload struct {
	Address      string         `json:"address"`
	Type         string         `json:"type"`
	Name         string         `json:"name"`
	Module       string         `json:"module,omitempty"`
	Provider     string         `json:"provider"`
	Actions      []string       `json:"actions"`
	Severity     string         `json:"severity"`
	ActionReason string         `json:"action_reason,omitempty"`
	Before       map[string]any `json:"before,omitempty"`
	After        map[string]any `json:"after,omitempty"`
}

// OutputPayload describes a Terraform output change. Output values are never
// sent — only the name, the actions, and the sensitive flag.
type OutputPayload struct {
	Name      string   `json:"name"`
	Actions   []string `json:"actions"`
	Sensitive bool     `json:"sensitive"`
}

// BuildPayload constructs a Payload from an analyzed plan. The analysis is the
// single source of truth: exactly the resources it contains (post filtering,
// data sources and no-ops already excluded) are sent to the model, so every
// resource on the wire carries a computed severity. criticalResources must be
// the effective list the analyzer classified with — defaults merged with user
// config — not the raw config value.
func BuildPayload(analysis *plan.PlanAnalysis, criticalResources []string) *Payload {
	resources := make([]ResourcePayload, 0, len(analysis.ResourceChanges))
	for _, r := range analysis.ResourceChanges {
		beforeSensitive := parseSensitive(r.Change.BeforeSensitive)
		afterSensitive := parseSensitive(r.Change.AfterSensitive)

		before := sanitizeAndDiff(r.Change.Before, r.Change.After, beforeSensitive)
		after := sanitizeAndDiff(r.Change.After, r.Change.Before, afterSensitive)

		resources = append(resources, ResourcePayload{
			Address:      r.Address,
			Type:         r.Type,
			Name:         r.Name,
			Module:       r.ModuleAddress,
			Provider:     r.Provider,
			Actions:      r.Actions,
			Severity:     string(r.Severity),
			ActionReason: r.ActionReason,
			Before:       before,
			After:        after,
		})
	}

	outputs := make([]OutputPayload, 0, len(analysis.OutputChanges))
	for _, oc := range analysis.OutputChanges {
		outputs = append(outputs, OutputPayload{
			Name:      oc.Name,
			Actions:   oc.Actions,
			Sensitive: oc.Sensitive,
		})
	}

	return &Payload{
		TerraformVersion: analysis.TerraformVersion,
		Counts: Counts{
			Total:         analysis.TotalChanges,
			Additions:     analysis.Additions,
			Modifications: analysis.Modifications,
			Deletions:     analysis.Deletions,
			Replacements:  analysis.Replacements,
		},
		CriticalResources: criticalResources,
		Resources:         resources,
		OutputChanges:     outputs,
	}
}

// MarshalCompact returns the compact JSON encoding of the payload, suitable
// for inclusion as the user message body.
func (p *Payload) MarshalCompact() ([]byte, error) {
	return json.Marshal(p)
}

// parseSensitive decodes Terraform's before_sensitive/after_sensitive field,
// which is either a bool (the entire value is/isn't sensitive) or a nested
// object mirroring the value's shape with bool leaves. Returns nil when the
// field is absent or false.
//
// The wire type is json.RawMessage because the JSON schema is polymorphic;
// once `terraform-json` adoption lands (tracked in roadmap.md) this helper
// becomes a 1-line type assertion.
func parseSensitive(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	// Bool false means "nothing sensitive"; nil signals callers to skip work.
	if b, ok := v.(bool); ok && !b {
		return nil
	}
	return v
}

// sanitizeAndDiff returns the subset of `self` whose values differ from `other`,
// with any attribute marked sensitive (via the matching sensitivity tree)
// replaced by redactedMarker. Keys with values equal to the matching `other`
// entry are dropped — the model sees only attributes whose value actually
// changed, mirroring Terraform's own diff display.
//
// Nested maps recurse; nested slices are compared as opaque JSON. The depth of
// real-world plan attributes is shallow enough that this is fine.
func sanitizeAndDiff(self, other map[string]any, sensitive any) map[string]any {
	if len(self) == 0 {
		return nil
	}
	if wholeSensitive(sensitive) {
		return redactedDiff(self, other)
	}

	sensitiveMap, isMap := sensitive.(map[string]any)
	if !isMap {
		sensitiveMap = nil
	}
	out := make(map[string]any)
	for k, v := range self {
		if other != nil {
			if ov, ok := other[k]; ok && jsonEqual(v, ov) {
				continue
			}
		}
		out[k] = sanitizeValue(v, sensitiveMap[k])
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// wholeSensitive reports whether the entire value (not just nested keys) is
// marked sensitive — Terraform encodes this as the literal JSON value `true`
// in the *_sensitive field.
func wholeSensitive(sensitive any) bool {
	b, ok := sensitive.(bool)
	return ok && b
}

// redactedDiff returns the changed keys with all values replaced by
// redactedMarker. Unchanged keys are still dropped so we don't ship
// "<redacted>" for things that didn't actually change.
func redactedDiff(self, other map[string]any) map[string]any {
	out := make(map[string]any, len(self))
	for k, v := range self {
		if other != nil {
			if ov, ok := other[k]; ok && jsonEqual(v, ov) {
				continue
			}
		}
		out[k] = redactedMarker
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// sanitizeValue applies redaction + truncation to a single value, recursing
// into nested maps. The sensitivity argument is the matching subtree from
// before_sensitive/after_sensitive (or nil if no sensitivity at this path).
func sanitizeValue(v, sensitive any) any {
	if wholeSensitive(sensitive) {
		return redactedMarker
	}
	switch typed := v.(type) {
	case map[string]any:
		sensitiveMap, isMap := sensitive.(map[string]any)
		if !isMap {
			sensitiveMap = nil
		}
		out := make(map[string]any, len(typed))
		for k, child := range typed {
			out[k] = sanitizeValue(child, sensitiveMap[k])
		}
		return out
	case string:
		if len(typed) > maxAttrValueChars {
			return typed[:maxAttrValueChars] + truncationSuffix
		}
		return typed
	default:
		return v
	}
}

// jsonEqual reports whether two arbitrary decoded-JSON values are equal. It
// uses JSON re-marshaling for deep equality without pulling in reflect.DeepEqual's
// surprises (e.g. nil-vs-empty-slice distinctions that don't matter here).
func jsonEqual(a, b any) bool {
	aj, errA := json.Marshal(a)
	bj, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	return string(aj) == string(bj)
}
