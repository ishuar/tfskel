---
name: fail-fast
description: "Remove defensive checks and silent fallbacks that only fire when upstream is broken"
---

Scan the code in scope (current changes by default, or files/paths given as args) and remove defensive code that hides bugs.

## What to remove

- **Nil / empty-string checks on values upstream always populates.** If the caller guarantees `x` is set, `if x == nil` and `if x == ""` inside the callee is dead defensive code.
- **Double fallbacks** (`if x == "" { x = y }; if x == "" { x = z }`). Usually a sign the author didn't trust their own invariants.
- **Re-validation of arguments** passed between internal functions — the caller and type system already guarantee them.
- **Silent recovery from "impossible" cases.** Returning a default when the invariant is violated hides the underlying bug instead of surfacing it.

## What to keep

- **Checks at system boundaries:** CLI flag parsing, file I/O errors, external API responses, user-provided input.
- **Genuine conditional rendering / logic** based on real optionality (e.g., `if len(items) > 0` when the field is legitimately optional in config).
- **Untrusted-pointer checks** before dereferencing values from external sources.

## Process

1. Scope to the right code. If the user didn't name files, run `git diff` / `git status` and work on what's changed.
2. For each check you find, ask: *"If I delete this, what changes in production behavior?"*
   - Answer = "nothing, it only matters when something upstream is broken" → **delete**.
   - Answer = "real user-facing behavior changes" → **keep**.
3. Where the invariant being relied on is non-obvious, add a one-line comment naming the upstream guarantee (e.g., `// populated by the runner before this is called`). Don't over-comment.
4. Prefer loud failure over silent fallback when an invariant *must* be enforced: explicit error return, or panic for genuine programming-bug conditions.
5. Run the test suite after cleanup. Anything that breaks was relying on the removed behavior — investigate whether the test was defensive too, or whether the invariant is weaker than assumed.

## Anti-patterns to watch for

- "Just in case" comments — usually the sign of a check that should be deleted.
- Checks that duplicate type-system or caller guarantees.
- Logging-and-continuing when the "impossible" case fires — this is a silent fallback with extra noise.

## Rule of thumb

> If deleting a check would only change behavior when something upstream is broken, delete the check.

Report what you removed and why, briefly. Don't restate the rule of thumb in the report — show the concrete before/after reasoning per site.
