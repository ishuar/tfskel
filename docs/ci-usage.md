# Using tfskel in GitHub Actions

tfskel touches CI in two ways. It can generate a set of GitHub Actions workflows for you (`tfskel init --workflows` plus `tfskel scaffold workflows --env <env>`), and it can run inside any workflow you write yourself (`tfskel validate`, `tfskel review plan`). This page covers both: what the generated workflows do and expect, how to install and call tfskel in your own jobs, how color and machine output behave under CI, and which exit codes to gate on. Flags are not repeated here; see the [command reference](reference/tfskel.md).

## Generated workflows

`tfskel init --workflows` (or `workflows.create: true` in `.tfskel.yaml`) copies four shared files into `.github/workflows/`. They are static: the same bytes ship in every tfskel binary and nothing in them is rendered from config.

| File | Trigger | Role |
|---|---|---|
| `lint.yaml` | `pull_request` to `main`, `workflow_dispatch` | Lints every changed root module |
| `reusable-detect-changes.yaml` | `workflow_call` | Turns changed files into a job matrix |
| `reusable-lint.yaml` | `workflow_call` | fmt, tflint, optional terraform-docs for one directory |
| `reusable-terraform-plan-apply.yaml` | `workflow_call` | Plan, review with tfskel, gated apply for one directory |

`tfskel scaffold workflows --env <env>` then adds one caller workflow per environment that wires `reusable-detect-changes.yaml` to `reusable-terraform-plan-apply.yaml`.

### How changed directories are detected

`reusable-detect-changes.yaml` takes one input, `terraform_files_pattern` (for example `envs/dev/**/*.tf`), runs `tj-actions/changed-files` with `dir_names: "true"`, and enriches each directory with `jq`:

```bash
{path: ., region: split("/")[2], environment: split("/")[1]}
```

The split assumes the `envs/<env>/<region>/<app>` layout that `tfskel scaffold` creates. Its `matrix` output feeds a `strategy.matrix.include` in the callers, so each changed root module gets its own job. If nothing changed, the matrix is `[]` and the downstream job is skipped.

### `lint.yaml` and `reusable-lint.yaml`

`lint.yaml` runs on pull requests to `main` that touch `envs/**` or the lint workflows themselves. On `workflow_dispatch` it accepts `terraform_files_path` (default `envs/**/*.tf`). For each changed directory it calls `reusable-lint.yaml`, whose inputs are:

| Input | Default | Purpose |
|---|---|---|
| `terraform_files_path` | required | Directory to lint |
| `enable_terraform_docs_check` | `false` | Run terraform-docs and fail on README diff |
| `terraform_docs_version` | `v0.20.0` | terraform-docs release |
| `tflint_version` | `v0.59.1` | tflint release |

The job reads the Terraform version from `envs/<env>/.terraform-version`, runs `terraform fmt -check -recursive -diff`, then `tflint -f compact`, each with `continue-on-error`. A summary table is posted as a PR comment only when something failed and deleted again once all checks pass. The last step exits 1 if any check failed. tfskel itself is not run by the lint workflow.

### `reusable-terraform-plan-apply.yaml`

Inputs, quoted from the template:

| Input | Required | Default | Purpose |
|---|---|---|---|
| `terraform_files_path` | yes | | Root module directory |
| `aws_region` | yes | | Region passed to `configure-aws-credentials` |
| `aws_role_to_assume` | yes | | IAM role ARN for OIDC |
| `environment` | yes | | GitHub environment used by the apply job |
| `auto_apply` | no | `false` | Apply on PR and manual dispatch |
| `terraform_log_level` | no | `INFO` | `TF_LOG` |
| `runner` | no | `ubuntu-latest` | `runs-on` |
| `cache_hash_file` | no | `/versions.tf` | File hashed for the provider cache key |
| `tfskel_version` | no | `latest` | Version installed for the review step |
| `tfskel_force_color` | no | `true` | Sets `FORCE_COLOR` for tfskel |

Secrets: `terraform_modules_ssh_deploy_key` (SSH key for private module sources) and `terraform_extra_secrets` (one `KEY=VALUE` per line, appended to `GITHUB_ENV`). The workflow declares `permissions: pull-requests: write, contents: read, id-token: write`.

AWS auth uses no static credentials. Both jobs call `aws-actions/configure-aws-credentials` with `role-to-assume: ${{ inputs.aws_role_to_assume }}`, so the role must trust the GitHub OIDC provider for your repository.

The `plan` job runs `dflook/terraform-plan` with `add_github_comment: true`. When that step reports `changes == 'true'`, the next step installs tfskel and reviews the plan:

```bash
[[ "${TFSKEL_VERSION}" == "latest" ]] && unset TFSKEL_VERSION
curl -fsSL https://raw.githubusercontent.com/ishuar/tfskel/main/install.sh | bash
tfskel review plan --json-file ${{ steps.plan.outputs.json_plan_path }}
```

`INSTALL_DIR` is `/usr/local/bin`. `tfskel_version: latest` means the newest release; any other value is passed to `install.sh` as `TFSKEL_VERSION`. The review runs without `--detailed-exitcode`, so it is informational: it prints the summary into the job log and never fails the plan job.

The `apply` job needs `plan`, runs in `environment: ${{ inputs.environment }}`, and only starts when the plan has changes and one of these holds:

- `pull_request` and `auto_apply == true`
- `push` to `refs/heads/main`
- `workflow_dispatch` and `auto_apply == true`

A push to `main` always applies, regardless of `auto_apply`. Put required reviewers on the GitHub environment and branch protection on `main` if that is too eager for you. `dflook/terraform-apply` re-plans and compares against the hash stored in the PR comment, so only the reviewed plan is applied.

### Per-environment caller workflow

`tfskel scaffold workflows --env dev` renders `terraform-plan-apply.yaml.tmpl` into `.github/workflows/<env>-<name>.yaml`:

- `name` is `workflows.name` from `.tfskel.yaml`; the default is `terraform-plan-apply`, so you get `dev-terraform-plan-apply.yaml`.
- `workflows.name` is a plain string. Values containing `{{` or `}}` are rejected. A trailing `.yaml` is stripped before the extension is added back.
- The final file name must not contain `..`, `/`, or `\`, and must survive `filepath.Clean` unchanged; otherwise generation fails.
- `workflows.create: false` disables this command entirely.

The caller triggers on `pull_request` and `push` to `main` for `envs/<env>/**` and for the workflow file itself, and on `workflow_dispatch` with inputs `auto_apply` (default `false`) and `tfskel_version` (default `latest`). It calls `reusable-detect-changes.yaml` with `envs/<env>/**/*.tf`, then fans out to `reusable-terraform-plan-apply.yaml` with `terraform_files_path`, `aws_region`, and `environment` taken from the matrix.

`aws_role_to_assume` is baked in at scaffold time from `workflows.aws_role_arn`, or from `arn:aws:iam::<account_id>:role/<workflows.aws_role_name>` using the account mapped to that environment. If neither is set the file contains `REPLACE_WITH_ROLE_TO_ASSUME`; edit it before the first run. The `secrets:` block for the SSH deploy key is left commented out.

tfskel does not check that the shared files exist. The caller references them with `uses: ./.github/workflows/reusable-...`, so GitHub fails the run if `tfskel init --workflows` was never executed. The order is: `init --workflows`, then `scaffold workflows` once per environment.

## Running tfskel in your own workflow

### Install

```yaml
- name: Install tfskel
  env:
    TFSKEL_VERSION: "<RELEASED_VERSION>"
    INSTALL_DIR: /usr/local/bin
  run: curl -fsSL https://raw.githubusercontent.com/ishuar/tfskel/main/install.sh | bash
```

`install.sh` downloads the release tarball for the runner's OS and architecture (Linux or Darwin, `x86_64` or `arm64`), verifies it against `checksums.txt`, and places the binary in `INSTALL_DIR` (default `~/.local/bin`, which is not on `PATH` on GitHub runners). Leave `TFSKEL_VERSION` unset for the latest release; a value must be `MAJOR.MINOR.PATCH` with or without a `v` prefix. The script needs `curl`, `tar`, and `sha256sum` or `shasum`.

With a Go toolchain on the runner, `go install github.com/ishuar/tfskel@<RELEASED_VERSION>` works too; the binary lands in `$(go env GOPATH)/bin`.

### Color

tfskel decides on color in this order, first match wins:

1. `NO_COLOR` set (any value): off.
2. `FORCE_COLOR` set to anything other than `0` or `false`: on.
3. `CI` equals `true` or `1`: off.
4. Otherwise `--no-color` decides.

GitHub Actions sets `CI=true`, so output is plain by default. Set `FORCE_COLOR=1` on the step to get ANSI colors in the job log.

### Machine-readable output

`tfskel validate` and `tfskel review plan` accept `--format json` and `--format csv`. In those modes every log line moves to stderr and colors are disabled, so stdout contains only the document:

```yaml
- run: tfskel review plan --json-file plan.json --format json > review.json
```

`--ai` is only accepted with `--format table`.

## Exit codes

Every command exits `1` on an error, so gate on the specific code, not on `!= 0`.

| Command | 0 | 1 | 2 | 3 |
|---|---|---|---|---|
| `tfskel validate` | all checks pass | error | findings (drift, tool issues) | |
| `tfskel review plan` | review completed | error | | |
| `tfskel review plan --detailed-exitcode` | no changes | error | additions or modifications | deletions or replacements |
| `init`, `scaffold`, `scaffold workflows` | done | error | | |

Details live in the [validate](reference/tfskel_validate.md) and [review plan](reference/tfskel_review_plan.md) references.

### Pattern 1: fail the job

```yaml
- name: Validate repo
  run: tfskel validate

- name: Review plan
  run: tfskel review plan --json-file plan.json --detailed-exitcode
```

Any non-zero exit stops the job. For `review plan` this fails whenever the plan has changes, which is usually only wanted in a drift-detection job.

### Pattern 2: report, then continue

```yaml
- name: Review plan
  id: review
  continue-on-error: true
  run: tfskel review plan --json-file plan.json --detailed-exitcode | tee review.md

- name: Comment on PR
  if: steps.review.outcome == 'failure' && github.event_name == 'pull_request'
  env:
    GH_TOKEN: ${{ github.token }}
  run: gh pr comment ${{ github.event.pull_request.number }} --body-file review.md

- name: Fail on destructive changes
  if: steps.review.outcome == 'failure'
  run: exit 1
```

`continue-on-error: true` keeps the job green while `steps.review.outcome` still reflects the real result. Drop the last step and the job applies after commenting. To distinguish 2 from 3, capture `$?` into `$GITHUB_OUTPUT` in the review step instead of relying on `outcome`. Commenting needs `permissions: pull-requests: write`.

## `--ai` in CI

`tfskel review plan --ai` appends a written analysis below the table using Anthropic (default, `ANTHROPIC_API_KEY`) or Gemini (`TFSKEL_AI_PROVIDER=gemini`, `GEMINI_API_KEY`). Pass the key as a step `env:` from a repository secret. The AI step is additive: a missing key, a network error, or a bad model name logs a warning and the command exits with whatever code the plan analysis produced, so forks without the secret still pass. Provider, model, and token options are in [ai-review.md](ai-review.md).
