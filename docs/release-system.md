# Automated Release System

## Overview

tfskel uses **Google's release-please** for fully automated version management, changelog generation, and binary distribution following Semantic Versioning and Go best practices.

**What this means**: Commit with conventional format → Push to main → Release PR created automatically → Merge PR → Release published with binaries.

## Quick Start

### 1. Commit Using Conventional Format

```bash
# Feature (minor bump: 0.1.0 → 0.2.0)
git commit -m "feat: add drift detection for module versions"

# Bug fix (patch bump: 0.1.0 → 0.1.1)
git commit -m "fix: handle nil pointer in template renderer"

# Breaking change (major bump: 0.1.0 → 1.0.0)
git commit -m "feat!: redesign CLI interface

BREAKING CHANGE: The --output flag has been renamed to --format"
```

### 2. Push to Main

```bash
git push origin main
```

### 3. Review & Merge Release PR

- **Release Please** creates/updates a release PR automatically
- PR includes updated `CHANGELOG.md`, `cmd/version.go`, and manifest
- Review the changes, then merge

### 4. Binaries Published Automatically

- Merging the release PR triggers a GitHub release
- **GoReleaser** builds and publishes binaries for all platforms
- Users can install: `go install github.com/ishuar/tfskel@latest`

## Conventional Commit Format

**Structure**:
```
<type>[optional scope]: <description>

[optional body]

[optional footer]
```

**Types & Version Bumps**:

| Type | Version Change | Example |
|------|----------------|---------|
| `feat:` | **Minor** (0.1.0 → 0.2.0) | New features |
| `fix:` | **Patch** (0.1.0 → 0.1.1) | Bug fixes |
| `perf:` | **Patch** (0.1.0 → 0.1.1) | Performance improvements |
| `refactor:` | **Patch** (0.1.0 → 0.1.1) | Code refactoring |
| `feat!:` or `BREAKING CHANGE:` | **Major** (0.1.0 → 1.0.0) | Breaking changes |
| `docs:`, `test:`, `ci:`, `chore:` | **None** | No version bump |

**Examples**:

```bash
# Feature with description
git commit -m "feat(drift): add support for provider version comparison

Extends drift detection to compare provider versions across workspace.
Includes both AWS and Kubernetes providers."

# Bug fix
git commit -m "fix: correct region name transformation for ap-south-1"

# Breaking change method 1 (exclamation mark)
git commit -m "feat!: change config file structure"

# Breaking change method 2 (footer)
git commit -m "feat: redesign template system

BREAKING CHANGE: Templates now use new directory structure.
Migration: Move custom templates from 'templates/' to 'custom-templates/'"

# Multiple scopes
git commit -m "fix(generator,templates): handle edge cases in template rendering"
```

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│  Developer: Commit with conventional format                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Push to main branch                                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Release Please Workflow (.github/workflows/release-please.yml) │
│  • Analyzes commits since last release                          │
│  • Calculates next version (SemVer)                             │
│  • Creates/Updates Release PR with:                             │
│    - Updated CHANGELOG.md                                       │
│    - Updated cmd/version.go                                     │
│    - Updated .release-please-manifest.json                      │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Maintainer: Review & Merge Release PR                          │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Release Please: Creates GitHub Release with Tag                 │
│  • Tag format: v0.2.0                                           │
│  • Release notes from CHANGELOG.md                              │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Release Workflow (.github/workflows/release.yml)               │
│  • Triggered by release publication                             │
│  • Runs GoReleaser to build binaries for:                       │
│    - Linux: amd64, arm64, 386                                   │
│    - macOS: amd64, arm64                                        │
│    - Windows: amd64, 386                                        │
│  • Generates SHA256 checksums                                   │
│  • Uploads binaries to GitHub release                           │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration Files

### `cmd/version.go` - Version Source of Truth
```go
package cmd

// Version is the semantic version of tfskel
// This value is automatically updated by release-please during releases
const Version = "0.0.1"
```

**Why in source code**:
- Go best practice: version embedded in binary
- Works with `go install`, `go get`, and `go version -m`
- No ldflags injection needed
- Single source of truth

### `release-please-config.json` - Release Configuration
```json
{
  "$schema": "https://raw.githubusercontent.com/googleapis/release-please/main/schemas/config.json",
  "release-type": "go",
  "bump-minor-pre-major": true,
  "bump-patch-for-minor-pre-major": true,
  "include-component-in-tag": false,
  "include-v-in-tag": true,
  "extra-files": [
    "cmd/version.go"
  ],
  "packages": {
    ".": {}
  }
}
```

**Key settings**:
- `release-type: go` - Go package release strategy
- `bump-minor-pre-major: true` - Allow 0.x.0 releases
- `extra-files: ["cmd/version.go"]` - Update version in source code
- `include-v-in-tag: true` - Tag format: v0.1.0

### `.release-please-manifest.json` - Current Version
```json
{
  ".": "0.0.1"
}
```

**Automatically maintained** by release-please. Do not edit manually.

### `.goreleaser.yaml` - Binary Build Configuration
```yaml
builds:
  - id: tfskel
    binary: tfskel
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64, "386"]
    ldflags:
      - -s -w
      - -X github.com/ishuar/tfskel/cmd.Commit={{.Commit}}
      - -X github.com/ishuar/tfskel/cmd.Date={{.Date}}
      - -X github.com/ishuar/tfskel/cmd.BuildTime={{.CommitTimestamp}}
    mod_timestamp: "{{.CommitTimestamp}}"
    flags:
      - -trimpath
```

**Note**: Version is NOT injected via ldflags (comes from `cmd/version.go`)

## GitHub Actions Workflows

### `.github/workflows/release-please.yml`

**Trigger**: Every push to `main` branch
**Purpose**: Create/update Release PR

**Actions**:
```yaml
on:
  push:
    branches: [main]

jobs:
  release-please:
    runs-on: ubuntu-latest
    steps:
      - uses: googleapis/release-please-action@v4
        with:
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json
```

### `.github/workflows/release.yml`

**Trigger**: When a GitHub release is published
**Purpose**: Build and upload binaries

**Actions**:
```yaml
on:
  release:
    types: [published]

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: '1.24'
      - uses: goreleaser/goreleaser-action@v6
        with:
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Release Workflow in Detail

### Phase 1: Development (Continuous)

```bash
# Developer makes changes
git checkout -b feature/drift-detection
# ... make changes ...
git commit -m "feat(drift): add version comparison for providers"
git push origin feature/drift-detection

# Create PR, review, merge to main
```

### Phase 2: Release PR Creation (Automatic)

After merge to main, release-please workflow runs:

1. **Scans commits** since last release (`v0.0.1`)
2. **Finds** conventional commits: `feat:`, `fix:`, etc.
3. **Calculates** next version based on commit types
4. **Creates/Updates Release PR** titled "chore(main): release 0.1.0"

**Release PR contains**:
- `CHANGELOG.md` with new section for 0.1.0
- `cmd/version.go` with updated version constant
- `.release-please-manifest.json` with new version

### Phase 3: Release PR Review (Manual)

```bash
# Review the Release PR
# Check:
# - Is the version bump correct? (feat = minor, fix = patch)
# - Does CHANGELOG.md look good?
# - Are breaking changes properly documented?

# If good: Merge the Release PR
# If issues: Add more commits to main (Release PR auto-updates)
```

### Phase 4: Release Publication (Automatic)

On Release PR merge:

1. **Release-please creates**:
   - Git tag: `v0.1.0`
   - GitHub release with CHANGELOG content

2. **Release workflow triggers**:
   - GoReleaser builds binaries for all platforms
   - Archives include: binary, README, LICENSE, CONTRIBUTING, docs/
   - SHA256 checksums generated
   - All uploaded to GitHub release

### Phase 5: User Installation (Automatic)

```bash
# Users can now install
go install github.com/ishuar/tfskel@v0.1.0
go install github.com/ishuar/tfskel@latest

# Or download binaries from GitHub releases
```

## Common Scenarios

### Scenario 1: Regular Bug Fix

```bash
git commit -m "fix(generator): handle empty account mapping correctly

Previously crashed when account_mapping was empty.
Now returns helpful error message."
git push origin main
```

**Result**: Release PR for `v0.0.2` (patch bump)

### Scenario 2: New Feature

```bash
git commit -m "feat(templates): add custom function support

Users can now define custom template functions in config:
- dateFormat: Format dates
- mathCalc: Mathematical operations
- randomString: Generate random strings"
git push origin main
```

**Result**: Release PR for `v0.1.0` (minor bump)

### Scenario 3: Breaking Change

```bash
git commit -m "feat!: change configuration file structure

BREAKING CHANGE: Configuration format has changed.

Before:
  terraform_version: ~> 1.13
  provider: { ... }

After:
  version:
    terraform: ~> 1.13
  providers:
    aws: { ... }

Migration: Run 'tfskel config migrate' to convert existing files."
git push origin main
```

**Result**: Release PR for `v1.0.0` (major bump)

### Scenario 4: Multiple Changes

```bash
# Multiple commits since last release
git commit -m "feat: add JSON output format"
git commit -m "fix: correct CSV escaping"
git commit -m "docs: update README with new examples"
git push origin main
```

**Result**: Release PR for `v0.1.0` (highest bump wins: minor from feat:)

**CHANGELOG will include**:
- ✨ Features: add JSON output format
- 🐛 Bug Fixes: correct CSV escaping
- 📚 Documentation: update README with new examples

## Modifying the Release System

### Change Version Bumping Behavior

Edit `release-please-config.json`:

```json
{
  "release-type": "go",
  "bump-minor-pre-major": false,  // Only allow patch bumps before 1.0.0
  "bump-patch-for-minor-pre-major": false,
  "extra-files": ["cmd/version.go"]
}
```

### Add Changelog Sections

Release-please uses default Go changelog sections. To customize, add to `release-please-config.json`:

```json
{
  "release-type": "go",
  "changelog-sections": [
    {"type": "feat", "section": "✨ Features"},
    {"type": "fix", "section": "🐛 Bug Fixes"},
    {"type": "perf", "section": "⚡ Performance"},
    {"type": "refactor", "section": "♻️ Refactoring"},
    {"type": "docs", "section": "📚 Documentation"},
    {"type": "test", "section": "✅ Tests"},
    {"type": "build", "section": "🔧 Build System"},
    {"type": "ci", "section": "👷 CI/CD"},
    {"type": "chore", "section": "🧹 Chores", "hidden": false}
  ],
  "extra-files": ["cmd/version.go"]
}
```

### Add More Version Files

To update version in multiple files (e.g., Helm chart):

```json
{
  "extra-files": [
    "cmd/version.go",
    "charts/tfskel/Chart.yaml",
    "pkg/version/version.go"
  ]
}
```

### Change Binary Build Platforms

Edit `.goreleaser.yaml`:

```yaml
builds:
  - goos: [linux, darwin, windows, freebsd]  # Add FreeBSD
    goarch: [amd64, arm64, "386", arm]       # Add ARM
```

### Add Pre-release Versions

For alpha/beta releases:

```bash
# Manual pre-release creation
git tag v0.2.0-alpha.1
git push origin v0.2.0-alpha.1

# Or use release-please with prerelease config
```

## Troubleshooting

### Release PR Not Created

**Problem**: No PR appears after pushing commits
**Solutions**:
1. Check commit format: Must use `feat:`, `fix:`, etc.
2. Verify GitHub Actions has write permissions
3. Check workflow logs: Actions → Release Please
4. Ensure commits exist since last release

### Wrong Version Bump

**Problem**: Expected minor bump but got patch
**Solutions**:
1. Use `feat:` for features (minor bump)
2. Use `fix:` for bugs (patch bump)
3. Use `feat!:` or `BREAKING CHANGE:` for breaking (major bump)
4. Check commit message follows conventional format exactly

### Version Not Updated in Code

**Problem**: `cmd/version.go` not updated
**Solutions**:
1. Verify file exists: `cmd/version.go`
2. Check `release-please-config.json` has correct path in `extra-files`
3. Ensure file has exact format: `const Version = "0.0.1"`
4. Review release-please PR for errors

### Binaries Not Published

**Problem**: Release created but no binaries
**Solutions**:
1. Check release workflow ran: Actions → Release
2. Verify all tests pass
3. Check GoReleaser logs for errors
4. Ensure release was "published" (not draft)

### Changelog Missing Commits

**Problem**: Some commits not in CHANGELOG.md
**Solutions**:
1. Ensure commits use conventional format
2. Check commit type is not excluded (e.g., `chore:` may be hidden)
3. Verify commits are in the commit range (between releases)

## Best Practices

### ✅ Do's

1. **Always use conventional commits** - Every commit should follow the format
2. **Write descriptive commit messages** - They become changelog entries
3. **Review Release PR carefully** - It shows exactly what will be released
4. **Keep commits atomic** - One logical change per commit
5. **Use scopes** - `feat(drift):`, `fix(generator):` for better organization
6. **Document breaking changes** - Always add migration guide

### ❌ Don'ts

1. **Don't edit manifest manually** - `.release-please-manifest.json` is auto-maintained
2. **Don't edit CHANGELOG.md directly** - Generated automatically
3. **Don't create tags manually** - Release-please handles this
4. **Don't bypass Release PR** - Always merge the PR, don't force-push
5. **Don't inject version via ldflags** - Use source code version instead

## Manual Release (Emergency Only)

If automated system fails, create release manually:

```bash
# 1. Update version manually
sed -i 's/Version = "0.0.1"/Version = "0.1.0"/' cmd/version.go

# 2. Update manifest
echo '{".":\n"0.1.0"}' > .release-please-manifest.json

# 3. Update CHANGELOG.md manually

# 4. Commit and create tag
git commit -am "chore: release 0.1.0"
git tag v0.1.0
git push origin main --tags

# 5. Create GitHub release manually from tag
# GoReleaser will trigger automatically
```

**Note**: After manual release, next automated release will work normally.

## References

- [Release Please Documentation](https://github.com/googleapis/release-please)
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Releases Guide](https://docs.github.com/en/repositories/releasing-projects-on-github)

---

**System Status**: ✅ Active and Automated
**Current Version**: 0.0.1
**Next Release**: Automatic on next conventional commit to main
