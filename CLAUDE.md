# CLAUDE.md

## Project

tfskel is a Go CLI tool for making terraform operations Simplified and predictable.

## Commit & PR conventions

Use [Conventional Commits](https://www.conventionalcommits.org/): `type(scope): description`

**Types:** feat, fix, perf, docs, test, refactor, build, ci, chore, style

**Scopes** (pick one, or omit for cross-cutting changes):

| Scope      | Covers                                               |
| ---------- | ---------------------------------------------------- |
| `init`     | `tfskel init` command and supporting code            |
| `scaffold` | `tfskel scaffold` command and supporting code        |
| `validate` | `tfskel validate` command and `internal/validate/`   |
| `review`   | `tfskel review` command and `internal/plan/`         |
| `config`   | Configuration system (`internal/config/`)            |

Cross-cutting changes (format, logger, build, CI) use the type alone — no scope needed.

Breaking changes: add `!` after the type, e.g. `feat(validate)!: redesign report output`

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

### Don't extract helpers just to reduce repetition

A helper earns its keep by doing one of two things:

1. **Hiding real complexity** — the inline version is hard to read or easy to get wrong.
2. **Centralizing a decision likely to change** — so updating it is one edit instead of many.

If a helper does neither — if it only shortens call sites by forwarding the same arguments to another function — inline the code. Repetition of short, obvious code is cheaper than the cognitive cost of jumping to a helper to find out it does nothing.

Red flags that a helper is pulling its weight:

- Its body is a single call with the same args rearranged.
- Its name paraphrases what the underlying function already says (`newCmdLogger` wrapping `logger.NewWithOptions(...)`).
- Callers lose useful information by not seeing the inline code (e.g. a dry-run wrap is load-bearing behavior and benefits from being visible at the call site).

"Four sites → one helper" is not an automatic win. Four readable inline calls beat one helper that forces every reader to jump elsewhere to understand a single-purpose call site.

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
