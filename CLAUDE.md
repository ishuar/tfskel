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

### Defensive code / fail fast

Trust invariants set by upstream code. Add runtime checks **only at system boundaries** (CLI flags, user input, file I/O, external APIs).

- Don't re-validate arguments passed between internal functions — the caller and the type system already guarantee them.
- Don't add silent fallbacks (`if x == "" { x = y }`) for cases that only fire when upstream is broken. They hide bugs.
- If an "impossible" case could theoretically occur, let it surface: explicit error, panic for programming bugs, or just ugly output. Visible failure beats silent recovery.

**Rule of thumb:** if deleting a check would only change behavior when something upstream is broken, delete the check.

### CLI Error Outputs

#### Design Principle

| Error Type               | Shows Usage? | Reasoning                                       |
|--------------------------|--------------|-------------------------------------------------|
| Unknown flag             | No           | User knows the command, just mistyped a flag    |
| Missing required flag    | No           | User knows the command, just forgot a flag      |
| Mutually exclusive flags | No           | User knows both flags, just combined them wrong |
| Invalid flag value       | No           | User knows the flag, just got the value wrong   |
| Wrong number of args     | No           | User knows the command, just forgot an argument |
| Runtime errors           | No           | Nothing to do with CLI syntax                   |
| No subcommand given      | **Yes**      | User needs to discover available subcommands    |

**Rule of thumb:** Show usage when the user doesn't know what commands/flags exist. Suppress it when they clearly do but made a mistake. Users who need help will use `--help`.
