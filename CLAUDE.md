# CLAUDE.md

## Project

tfskel is a Go CLI tool for making terraform operations Simplified and predictable.

## Commit & PR conventions

Use [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`

**Types:** feat, fix, perf, docs, test, refactor, build, ci, chore, style

**Scopes** (pick one):

| Category | Scope      | Covers                                      |
| -------- | ---------- | ------------------------------------------- |
| Commands | `cmd`      | CLI commands (init, scaffold, diff, review) |
| Internal | `config`   | Configuration system                        |
| Internal | `template` | Template rendering                          |
| Internal | `generate` | Template generation                         |
| Internal | `plan`     | Plan parsing/analysis                       |
| Internal | `diff`     | Diff computation                            |

Breaking changes: add `!` after the type, e.g. `feat(cmd)!: redesign CLI interface`

## Build & test

```bash
make check    # fmt, vet, lint, test
make build    # build binary
```

## Code style

- Follow Effective Go
- See CONTRIBUTING.md for full guidelines
