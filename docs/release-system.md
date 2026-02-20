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
│  Job 1: Release Please                                          │
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
│  • Sets release_created=true output                             │
└────────────────────────────┬────────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│  Release Please Workflow (.github/workflows/release-please.yml) │
│  Job 2: GoReleaser (conditional: if release_created == true)    │
│  • Checks out code at the release tag                           │
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

**File**: [cmd/version.go](../cmd/version.go)

**Current version**: `0.2.0`

**Purpose**: Embeds the semantic version as a constant that is automatically updated by release-please during releases.

**Why in source code**:
- Go best practice: version embedded in binary
- Works with `go install`, `go get`, and `go version -m`
- No ldflags injection needed
- Single source of truth

### `release-please-config.json` - Release Configuration

**Configuration**: [release-please-config.json](../release-please-config.json)

**Key settings**:
- `release-type: go` - Go package release strategy
- `bump-minor-pre-major: true` - Allows 0.x.0 minor version releases before 1.0.0
- `extra-files: ["cmd/version.go"]` - Automatically updates version constant in source code
- `include-v-in-tag: true` - Creates tags with v prefix (v0.2.0)
- `draft: false` - Publishes releases immediately (not as drafts)
- `changelog-sections` - Custom sections with emojis (✨ Features, 🐞 Bug Fixes, etc.)
- Hidden sections: docs, style, chore, refactor, build, ci
- Visible sections: feat, fix, perf, revert, test

### `.release-please-manifest.json` - Current Version

**Configuration**: [.release-please-manifest.json](../.release-please-manifest.json)

**Purpose**: Tracks the current released version (0.2.0)

⚠️ **Automatically maintained** by release-please. **Do not edit manually.**

### `.goreleaser.yaml` - Binary Build Configuration

**Configuration**: [.goreleaser.yaml](../.goreleaser.yaml)

**Key build settings**:
- Platforms: Linux, macOS, Windows
- Architectures: amd64, arm64, 386
- CGO disabled for static binaries
- Optimization: `-s -w` flags for smaller binaries
- Injects: Commit hash, build date, and timestamp
- Includes: README, LICENSE, CONTRIBUTING, docs/

**Important**: Version is NOT injected via ldflags—it comes from [cmd/version.go](../cmd/version.go)

## GitHub Actions Workflows

### `.github/workflows/release-please.yml`

**Workflow**: [.github/workflows/release-please.yml](../.github/workflows/release-please.yml)

**Triggers**:
- Every push to `main` branch
- Manual dispatch via workflow_dispatch

**Purpose**: Create/update Release PR and build binaries when released

**Concurrency**: Prevents overlapping runs (cancel-in-progress: false)

**Jobs**:

#### Job 1: Release Please
- **Action**: `googleapis/release-please-action@v4`
- **Outputs**: `release_created`, `tag_name`, `version`
- Analyzes commits and creates/updates Release PR
- Creates GitHub release when PR is merged

#### Job 2: GoReleaser (Conditional)
- **Condition**: Only runs when `release_created == 'true'`
- **Go version**: Automatically detected from `go.mod`
- **GoReleaser**: Uses latest v2.x version
- Checks out code at the release tag
- Builds and uploads binaries to the GitHub release

**Workflow**: [.github/workflows/manual-release.yaml](../.github/workflows/manual-release.yaml)

**Trigger**: Manual workflow dispatch only

**Purpose**: (Re)attach binaries to an existing release tag

**Use cases**:
- GoReleaser failed during automatic release
- Need to rebuild binaries for an existing release
- Binary artifacts missing from a release

**Input required**: `git_tag` (e.g., v0.2.0)

**Process**:
- Checks out code at the specified tag
- Uses Go version from `go.mod`
- Runs GoReleaser to build all platform binaries
- Attaches/replaces assets on the existing GitHub release

**Usage**:
1. Go to Actions → Manual Release to attach binaries
2. Click "Run workflow"
3. Enter the existing tag (e.g., `v0.2.0`)
4. GoReleaser will build and attach binaries to that release

### `.github/workflows/test.yaml`

**Workflow**: [.github/workflows/test.yaml](../.github/workflows/test.yaml)

**Triggers**: Pull requests and pushes to `main` branch

**Purpose**: Run tests, linting, and build validation

**Path filters**: Skips workflow when only docs/markdown files change

**Jobs**:

#### Job 1: Test Matrix
- **Operating Systems**: Ubuntu, macOS
- **Go Versions**: 1.24, 1.25.5
- Race detector enabled (`-race`)
- Coverage reports generated
- Codecov upload (Ubuntu + Go 1.25.5 only)

#### Job 2: Lint
- Validates `go mod tidy` consistency
- `golangci-lint` v2.8.0 (5-minute timeout)
- Runs after tests complete

#### Job 3: Build
- Optimized binary build (`-ldflags="-w -s"`)
- Smoke tests: `--version` and `init` commands
- Runs after tests complete

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

1. **Scans commits** since last release (`v0.2.0`)
2. **Finds** conventional commits: `feat:`, `fix:`, etc.
3. **Calculates** next version based on commit types
4. **Creates/Updates Release PR** titled "chore(main): release 0.3.0"

**Release PR contains**:
- `CHANGELOG.md` with new section for 0.3.0
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

1. **Release-please (Job 1) creates**:
   - Git tag: `v0.3.0`
   - GitHub release with CHANGELOG content
   - Sets `release_created=true` output

2. **GoReleaser (Job 2) conditionally triggers**:
   - Only runs when `release_created == 'true'`
   - Checks out code at the release tag
   - Builds binaries for all platforms using Go version from go.mod
   - Archives include: binary, README, LICENSE, CONTRIBUTING, docs/
   - SHA256 checksums generated
   - All uploaded to GitHub release

### Phase 5: User Installation (Automatic)

```bash
# Users can now install
go install github.com/ishuar/tfskel@v0.3.0
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

**Result**: Release PR for `v0.2.1` (patch bump)

### Scenario 2: New Feature

```bash
git commit -m "feat(templates): add custom function support

Users can now define custom template functions in config:
- dateFormat: Format dates
- mathCalc: Mathematical operations
- randomString: Generate random strings"
git push origin main
```

**Result**: Release PR for `v0.3.0` (minor bump)

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

**Result**: Release PR for `v1.0.0` (major bump - first major release)

### Scenario 4: Multiple Changes

```bash
# Multiple commits since last release
git commit -m "feat: add JSON output format"
git commit -m "fix: correct CSV escaping"
git commit -m "docs: update README with new examples"
git push origin main
```

**Result**: Release PR for `v0.3.0` (highest bump wins: minor from feat:)

**CHANGELOG will include**:
- ✨ Features: add JSON output format
- 🐞 Bug Fixes: correct CSV escaping
- Documentation changes are hidden by default per changelog-sections config

## Modifying the Release System

### Change Version Bumping Behavior

**File**: [release-please-config.json](../release-please-config.json)

**To only allow patch bumps before 1.0.0**:
- Set `bump-minor-pre-major: false`
- Set `bump-patch-for-minor-pre-major: false`

This prevents `feat:` commits from creating 0.x.0 releases before reaching 1.0.0.

### Modify Changelog Sections

**File**: [release-please-config.json](../release-please-config.json)

**Current configuration**: Custom sections with emojis (✨ Features, 🐞 Bug Fixes, etc.)

**To modify**:
- Edit the `changelog-sections` array
- Set `"hidden": false` to show a section in CHANGELOG
- Set `"hidden": true` to hide it from release notes
- Add new sections with custom `type` and `section` title

**Currently visible**: feat, fix, perf, revert, test
**Currently hidden**: docs, style, chore, refactor, build, ci

### Add More Version Files

**File**: [release-please-config.json](../release-please-config.json)

**To update version in multiple files** (e.g., Helm charts, package files):
- Add file paths to the `extra-files` array
- Release-please will update version strings in all listed files
- Example: `["cmd/version.go", "charts/tfskel/Chart.yaml", "pkg/version/version.go"]`

### Change Binary Build Platforms

**File**: [.goreleaser.yaml](../.goreleaser.yaml)

**To add more platforms/architectures**:
- Add OS to `goos` array (e.g., `freebsd`, `openbsd`)
- Add architecture to `goarch` array (e.g., `arm`, `ppc64le`)
- GoReleaser will build for all combinations

**Current platforms**: Linux, macOS, Windows
**Current architectures**: amd64, arm64, 386

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
1. Verify file exists: [cmd/version.go](../cmd/version.go)
2. Check [release-please-config.json](../release-please-config.json) has correct path in `extra-files`
3. Ensure file has exact format: `const Version = "0.2.0"`
4. Review release-please PR for errors

### Binaries Not Published

**Problem**: Release created but no binaries
**Solutions**:
1. Check if GoReleaser job ran: Actions → [Release Please](.github/workflows/release-please.yml) → go-releaser job
2. Verify `release_created` output was `true`
3. Check GoReleaser logs for build/upload errors
4. If GoReleaser failed, use [manual-release workflow](../.github/workflows/manual-release.yaml):
   - Actions → Manual Release to attach binaries
   - Enter the tag name (e.g., `v0.2.0`)
   - Run workflow to rebuild and attach binaries

### Changelog Missing Commits

**Problem**: Some commits not in [CHANGELOG.md](../CHANGELOG.md)
**Solutions**:
1. Ensure commits use conventional format
2. Check commit type is not excluded via [changelog-sections](../release-please-config.json) (e.g., `chore:` is hidden)
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

1. **Don't edit manifest manually** - [.release-please-manifest.json](../.release-please-manifest.json) is auto-maintained
2. **Don't edit CHANGELOG.md directly** - [CHANGELOG.md](../CHANGELOG.md) is generated automatically
3. **Don't create tags manually** - Release-please handles this
4. **Don't bypass Release PR** - Always merge the PR, don't force-push
5. **Don't inject version via ldflags** - Use [source code version](../cmd/version.go) instead

## Manual Release (Emergency Only)

If the automated system fails completely, you can create a release manually:

**Steps**:
1. Update version in [cmd/version.go](../cmd/version.go) (e.g., `0.2.0` \u2192 `0.3.0`)
2. Update version in [.release-please-manifest.json](../.release-please-manifest.json)
3. Manually update [CHANGELOG.md](../CHANGELOG.md) with release notes
4. Commit changes: `git commit -am "chore: release 0.3.0"`
5. Create and push tag: `git tag v0.3.0 && git push origin main --tags`
6. Create GitHub release manually from the tag
7. Run [manual-release workflow](../.github/workflows/manual-release.yaml) with tag `v0.3.0` to attach binaries

\u26a0\ufe0f **Note**: After manual release, the next automated release will work normally.

## References

- [Release Please Documentation](https://github.com/googleapis/release-please)
- [Conventional Commits Specification](https://www.conventionalcommits.org/)
- [Semantic Versioning](https://semver.org/)
- [GoReleaser Documentation](https://goreleaser.com/)
- [GitHub Releases Guide](https://docs.github.com/en/repositories/releasing-projects-on-github)

## Testing Changes

### Test Workflow

**Workflow**: [.github/workflows/test.yaml](../.github/workflows/test.yaml)

Before commits reach `main` and trigger releases, they are validated by the test workflow:

**Runs on**:
- All pull requests to `main`
- All pushes to `main`
- Skips when only docs/markdown changed

**Test matrix**:
- Operating Systems: Ubuntu, macOS
- Go Versions: 1.24, 1.25.5

**Validation steps**:
1. **Tests**: Unit tests with race detector and coverage
2. **Linting**: golangci-lint v2.8.0 with strict checks
3. **Build**: Binary compilation and smoke tests
4. **Coverage**: Upload to Codecov (Ubuntu + Go 1.25.5)

**Quality gates**:
- All tests must pass on all platforms
- Linting must pass with no errors
- Binary must build and execute successfully

---

**System Status**: ✅ Active and Automated
**Current Version**: 0.2.0
**Next Release**: Automatic on next conventional commit to main

**Workflows**:
- ✅ [Release Please](../.github/workflows/release-please.yml) (automated + manual trigger)
- ✅ GoReleaser (conditional on release)
- ✅ [Manual Release](../.github/workflows/manual-release.yaml) (for binary reattachment)
- ✅ [Tests](../.github/workflows/test.yaml) (PR validation)
