## Description

<!-- Briefly describe what changed and why -->
<!-- Select ONE primary scope (follows Go CLI conventions and maps to codebase structure):

Commands (user-facing):
  - cmd        → CLI commands (init, scaffold, diff, review)

Internal Packages (component-based):
  - config     → Configuration system
  - templates  → Template rendering
  - generator  → Template generation
  - plan       → Plan parsing/analysis
  - format     → Output formatting
  - fs         → File system operations

Tooling & Infra:
  - build      → Build system/Makefile
  - ci         → GitHub Actions/CI
  - deps       → Go dependencies
  - test       → Test infrastructure

General:
  - docs       → Documentation only
  - chore      → Maintenance tasks

Examples:
  feat(cmd): add --dry-run flag to scaffold
  fix(plan): handle null values in JSON parsing
  build(deps): bump cobra to v1.8.0
  docs: update installation guide
-->


## Changes

<!-- User-visible changes (CLI behavior, output, flags, etc.) -->
-
-

## Breaking Changes

<!-- ⚠️ If this changes CLI flags, config format, output structure, or behavior, describe migration steps -->

**None** _(or describe)_

## Related Issue

Fixes #

## Testing

```bash
make check
# Additional validation performed:
# -
```

## Checklist

- [ ] Scope correctly identified in title: `type(scope): description`
- [ ] Breaking changes marked with `!` suffix if applicable: `feat(scope)!:`
- [ ] Documentation updated (if user-facing changes)
- [ ] Tests added/updated
- [ ] `make check` passes locally
