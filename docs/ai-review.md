# AI plan review

`tfskel review plan --ai` appends a written analysis of the plan below the structured table. tfskel builds a sanitized JSON view of the changed resources, sends it to Anthropic Claude or Google Gemini with a fixed system prompt, and streams the model's Markdown back to the terminal. The AI step is additive: it never changes the table, the exit code, or the behaviour of the command when it fails. This page is the long form of the [README section](../README.md#ai-summary-for-plan-reviews). Flags are listed in the [`review plan` reference](reference/tfskel_review_plan.md); this page explains what they do.

## How a run proceeds

1. Parse the plan JSON and analyze every resource change (severity, action reason, critical-resource match).
2. Apply `--filter-severity`, `--min-severity`, or `--filter-action` to the resource list.
3. Print the structured table from the filtered analysis.
4. Build the AI client. A missing key or unknown provider stops here with a warning.
5. Build the payload from the same filtered analysis, print the `## AI Analysis` header, and stream the model's response.
6. Derive the exit code from the analysis alone. Nothing in step 4 or 5 can change it.

Steps 4 and 5 only run with `--ai`. Without it the command is unchanged.

## What it produces

After the table, tfskel prints a `## AI Analysis` block:

```markdown
## AI Analysis

- **Provider:** anthropic
- **Model:** claude-sonnet-4-6

### Blast Radius & Downtime Risk
...
### Security Implications
...
### Rollback & Pre-apply Checks
...
```

The system prompt asks the model for exactly these three `###` sections, in this order, and nothing else:

| Section | What the prompt asks for |
| --- | --- |
| Blast Radius & Downtime Risk | Per address: the action and severity, what changes (citing diff field names), what breaks, and why. Prioritizes stateful deletes, forced replacements, `critical_resources` types, and `count`/`for_each` churn. |
| Security Implications | Per address: a one-line concrete risk and the attribute change behind it. Covers IAM scope, network exposure, encryption toggles, secret and key rotation. Ends with the line `_LLM analysis — not a substitute for tfsec / checkov / trivy._` |
| Rollback & Pre-apply Checks | Per address: reversibility (`trivial`, `requires backup`, or `one-way`), what to do before apply, and what state you are left in if apply fails. |

The prompt also tells the model to discuss only `critical` and `high` severities (mentioning `medium`/`low` only when they interact with a higher finding), to treat `action_reason` as authoritative over its own inference, to never speculate about `<redacted>` or `...<truncated>` values, to cite a concrete address or attribute in every bullet, and to write one sentence when a section has no findings.

Output is streamed. Both providers write each text chunk to the terminal as it arrives, so the analysis appears progressively instead of after the full response.

## Providers and credentials

The provider is chosen by one environment variable. API keys come from the environment only; they are never read from `.tfskel.yaml` or from flags.

| Variable | Values | Notes |
| --- | --- | --- |
| `TFSKEL_AI_PROVIDER` | `anthropic` (default when unset), `gemini` | Any other value is an error: `unknown AI provider: TFSKEL_AI_PROVIDER="x" (want "anthropic" or "gemini")`. There is no silent fallback. |
| `ANTHROPIC_API_KEY` | key | Required for `anthropic`. |
| `GEMINI_API_KEY` | key | Required for `gemini`. |

```bash
# Claude (default)
export ANTHROPIC_API_KEY=...
tfskel review plan --json-file tfplan.json --ai

# Gemini
export GEMINI_API_KEY=...
TFSKEL_AI_PROVIDER=gemini tfskel review plan --json-file tfplan.json --ai
```

## Models and token limits

| Setting | Flag | Config key | Default |
| --- | --- | --- | --- |
| Model | `--ai-model` | `ai.model` | `claude-sonnet-4-6` for Anthropic, `gemini-2.5-flash` for Gemini |
| Max response tokens | `--ai-max-tokens` | `ai.max_tokens` | `8192` for both providers |

Precedence is flag, then config, then the provider's built-in default. The config is read once at the CLI boundary; a non-empty `--ai-model` or a positive `--ai-max-tokens` replaces the config value before the client is built. An empty model or a zero `max_tokens` means "the provider's choice", so an Anthropic default never leaks into a Gemini run or vice versa.

Why these defaults:

- `gemini-2.5-flash` is chosen for broad free-tier availability. Gemini 2.5 Pro free-tier access is often restricted (projects see a limit of 0), so it is not safe as a default.
- `8192` tokens fits the three required sections with headroom for large plans. The earlier `4096` default routinely truncated mid-sentence. Output budgets are cheap per call; the only cost of a high ceiling is a longer wall-clock when the model uses it.
- For Gemini, tfskel sets the thinking budget to `0`. Gemini 2.5 models enable thinking by default and those tokens count against the output limit; on Flash this consumed most of a 4096 budget and cut the visible Markdown short.

Gemini takes the token limit as a 32-bit integer. A value outside that range is rejected with `max_tokens out of range for Gemini`.

```yaml
# .tfskel.yaml
ai:
  model: claude-opus-4-7
  max_tokens: 12000
```

## What is sent to the model

The user message is one JSON document built from the analyzed plan. It mirrors `terraform show -json` but is reduced to what a reviewer needs.

| Field | Content |
| --- | --- |
| `terraform_version` | Terraform version from the plan. |
| `counts` | `total`, `additions`, `modifications`, `deletions`, `replacements`. These are the totals for the whole plan, as shown in the table header. |
| `critical_resources` | The effective critical-resource list: tfskel's defaults merged with `critical_resources` from config. The same list drives severity classification, so the model and the table never disagree. |
| `resources[]` | One entry per changed resource: `address`, `type`, `name`, `module`, `provider`, `actions`, `severity`, `action_reason` (when Terraform gave one), `before`, `after`. |
| `output_changes[]` | One entry per changed output: `name`, `actions`, `sensitive`. Output values are never sent. |

Rules applied while building `before` and `after`:

1. **Only changed attributes.** A key whose value is JSON-equal in `before` and `after` is dropped from both. Nested maps are compared key by key; lists are compared as opaque JSON values.
2. **Sensitive attributes are redacted.** tfskel reads Terraform's `before_sensitive` and `after_sensitive` trees. Any leaf marked `true` is replaced with the string `<redacted>`. If the whole value is marked sensitive, every changed key is sent as `<redacted>`. Unchanged keys are still dropped, so you do not see `<redacted>` for things that did not change.
3. **Oversized values are truncated.** A string longer than 500 characters is cut to 500 and suffixed with `...<truncated>`, so a `user_data` blob or rendered template cannot eat the token budget on its own.
4. **Data sources and no-op changes are never included.** The analyzer excludes them before the payload exists.
5. **Filters apply first.** `--filter-severity`, `--min-severity`, and `--filter-action` narrow the analysis before the table is rendered and before the payload is built. The model sees exactly the resources the table shows.

The system prompt is the only other thing sent. It is a constant string; the plan and the critical-resource list always travel in the user message.

## Failure modes

Every AI failure is a `[WARN]` log line. The table has already been printed, and the exit code is whatever the structured analysis decided (`0`, or `2`/`3` with `--detailed-exitcode`).

| Situation | What you see |
| --- | --- |
| `ANTHROPIC_API_KEY` or `GEMINI_API_KEY` unset | `--ai requested but skipping AI analysis: ANTHROPIC_API_KEY is not set` (or the Gemini equivalent). No `## AI Analysis` header is printed. |
| Unknown `TFSKEL_AI_PROVIDER` value | `--ai requested but skipping AI analysis: unknown AI provider: ...`. |
| API error, network failure, invalid model name | `AI analysis failed: ...`. The header may already be on screen. |
| Response hit the token ceiling | The partial text stays on screen, followed by a Markdown note: `> **Output truncated** at the model's max-tokens ceiling. Re-run with --ai-max-tokens <higher value> (or set ai.max_tokens in your config) to see the full analysis.` plus a `[WARN]` line. |

Truncation is detected from the provider's stop reason (`max_tokens` on Anthropic, `MAX_TOKENS` on Gemini), not by guessing from the text. Raise `--ai-max-tokens` or `ai.max_tokens` and run again.

## Restrictions

`--ai` works only with the default `--format table`. Combining it with `--format json` or `--format csv` fails before the plan is read:

```text
--ai is only supported with --format=table (got --format=json)
```

The command exits `1` in that case, as for any invalid flag combination.

## Cost

The system prompt is sent on every call. For Anthropic it is marked with ephemeral cache control, so repeat runs inside the cache window bill that block at the reduced cached-input rate. The prompt is a constant for this reason: anything dynamic goes in the user message, which is not cached. tfskel does not configure any caching for Gemini; the system prompt is passed as a system instruction and billed at the normal rate.

## CI example

```yaml
- name: Review plan
  env:
    ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
  run: |
    terraform show -json tfplan.binary > tfplan.json
    tfskel review plan --json-file tfplan.json --ai --no-color
```

The same step with Gemini:

```yaml
- name: Review plan
  env:
    TFSKEL_AI_PROVIDER: gemini
    GEMINI_API_KEY: ${{ secrets.GEMINI_API_KEY }}
  run: tfskel review plan --json-file tfplan.json --ai --no-color
```

If the secret is missing or empty, the step prints the table, logs `--ai requested but skipping AI analysis: ANTHROPIC_API_KEY is not set`, and exits with the same code it would have without `--ai`. Add `--detailed-exitcode` if you want the job to fail on changes; that decision is still made from the table, never from the AI output.

## Not a scanner

The AI summary reasons about the diff it is given. It cannot see state, provider defaults, or the modules behind the plan, and it is told not to guess about redacted or truncated values. It does not replace `tfsec`, `checkov`, or `trivy`. Keep the scanners in the pipeline and treat the summary as a reviewer's note, not a gate.
