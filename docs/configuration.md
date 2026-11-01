# Configuration

tfskel reads one YAML file, `.tfskel.yaml`, for everything that is not a command-line flag: Terraform and provider version constraints, the AWS account per environment, the S3 state bucket, custom template settings, GitHub workflow settings, and plan-review tuning. `tfskel init` writes a seed file with placeholders; `tfskel scaffold` and `tfskel validate` refuse to run until the placeholders are replaced. This page lists every key, its default, and the validation rule behind each error message. The annotated reference copy is [`.tfskel.example.yaml`](../.tfskel.example.yaml).

## Where the file is found

1. `--config/-c <path>` (persistent flag on every command) is used as-is when given.
2. Otherwise tfskel looks for `.tfskel.yaml` in the current working directory only. Parent directories are not searched.

```bash
tfskel scaffold myapp --env dev --region eu-central-1            # ./.tfskel.yaml
tfskel scaffold myapp --env dev --region eu-central-1 -c infra/tfskel.yaml
```

`scaffold`, `scaffold workflows`, and `validate` fail when no file is found:

```text
.tfskel.yaml not found in /path/to/cwd; run from the project root, pass --config/-c, or run 'tfskel init' to create one
```

`review plan` does not require a file; it reads `critical_resources`, `top_resources_count`, and `ai.*` when a file is present and uses defaults otherwise. `init` reads `.tfskel.yaml` from the target directory (`--dir`, default cwd), not from `--config`.

## Precedence

Flag > config file > built-in default. Only these flags override config keys:

| Flag | Command | Overrides key |
| --- | --- | --- |
| `--templates-dir` | `scaffold` | `templates.dir` |
| `--s3-bucket-name` | `scaffold` | `backend.s3.bucket_name` |
| `--workflows` | `init` | `workflows.create` |
| `--top-resources-count` | `review plan` | `top_resources_count` |
| `--ai-model` | `review plan` | `ai.model` |
| `--ai-max-tokens` | `review plan` | `ai.max_tokens` (ignored when `<= 0`) |

Config keys are not read from environment variables. There is no `TFSKEL_*` prefix mapping for YAML keys. The environment variables tfskel does read are unrelated to the file: `TFSKEL_LOG_LEVEL`, `TFSKEL_AI_PROVIDER`, `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `NO_COLOR`, `FORCE_COLOR`, `CI`. See [`tfskel review plan`](reference/tfskel_review_plan.md) for the AI ones.

## Keys

Keys marked "template" accept Go template syntax (`{{.Env}}`, `{{.AccountID}}`, `{{.Region}}`, `{{.ShortRegion}}`, `{{.AppDir}}`). A value without `{{` is used verbatim. See [templates.md](templates.md) for the full placeholder list.

### Versions

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `terraform_version` | string | `~> 1.13` | Terraform `required_version` constraint written into `versions.tf`. `init` strips the operator to pin the tool version (`~> 1.13` becomes `1.13.0`); without a config file `init` uses `1.13.1`. |
| `provider.aws.version` | string | `~> 6.0` | AWS provider version constraint. |

### `provider.aws`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `account_mapping` | map `env -> account ID` | none | Required. Keys are environment names; `init` creates one `envs/<env>/` directory per key. Values must be 12-digit strings. Quote them so YAML keeps leading zeros. |
| `regions` | list of strings | `[]` | Used by `init` to create `envs/<env>/<region>/`. Falls back to `eu-central-1` with a warning when empty. `scaffold` takes the region from `--region`, not from this list. |
| `default_tags` | map | `{}` | Rendered into the provider `default_tags` block. Keys are lowercased (`ManagedBy` becomes `managedby`); nothing else is changed. |

### `backend.s3`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `bucket_name` | string, template | none | Required. S3 bucket for Terraform state. Rendered per scaffold run, so `tfstate-{{.Env}}-{{.AccountID}}` gives one bucket per environment. |

### `templates`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `dir` | string | none (embedded templates) | Directory of custom `.tmpl` files. A file with the same name as an embedded template replaces it. |
| `upgrade` | list of strings | `[]` (all eligible) | Template file names (`main.tf.tmpl`, not the rendered name) that `scaffold --upgrade` may re-render. Empty means every template with a source marker is eligible. |

### `workflows`

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `create` | bool | `false` | When `true`, `init` generates the shared workflow files without `--workflows`. |
| `name` | string | none | Plain string stem for the per-environment workflow file: `my-terraform` plus `--env dev` gives `dev-my-terraform.yaml`. Go template syntax is rejected. Unset gives `dev-terraform-plan-apply.yaml`. |
| `aws_role_arn` | string, template | none | Full IAM role ARN for OIDC auth. Takes priority over `aws_role_name`. |
| `aws_role_name` | string, template | none | Role name; tfskel builds `arn:aws:iam::<account-id>:role/<name>`. When neither key is set the ARN is `arn:aws:iam::<account-id>:role/REPLACE_WITH_ROLE_TO_ASSUME`. |

### Plan review

| Key | Type | Default | Notes |
| --- | --- | --- | --- |
| `critical_resources` | list of strings | `[]` | Resource types added to the built-in critical list (RDS, DynamoDB, S3, VPC, security groups, IAM, KMS, Secrets Manager, WAF, and more). Merged without duplicates. Changes to critical resources are reported as high severity. |
| `top_resources_count` | int | `10` | Rows in the "by resource type" and "by module" tables. `0` shows all rows. Negative values fall back to `10`. |
| `ai.model` | string | provider default | Model for `review plan --ai`. Anthropic default `claude-sonnet-4-6`; Gemini default `gemini-2.5-flash`. |
| `ai.max_tokens` | int | `8192` | Response token cap for both providers. Values `<= 0` are ignored. Gemini rejects values above `2147483647` with `max_tokens out of range for Gemini`. |

## What `tfskel init` writes

`init` creates `.tfskel.yaml` only when none exists. The seed contains:

```yaml
backend:
    s3:
        bucket_name: CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME
critical_resources: []
provider:
    aws:
        account_mapping:
            dev: REPLACE_WITH_YOUR_DEV_ACCOUNT_ID
            prd: REPLACE_WITH_YOUR_PRD_ACCOUNT_ID
            stg: REPLACE_WITH_YOUR_STG_ACCOUNT_ID
        default_tags:
            managed_by: terraform
        regions:
            - eu-central-1
        version: ~> 6.0
terraform_version: ~> 1.13
```

Keys are written in alphabetical order with four-space indentation because the seed is marshalled from a map; the order carries no meaning.

Edit before running `scaffold` or `validate`:

1. Every `account_mapping` value: replace with a 12-digit account ID. Delete environments you do not use.
2. `backend.s3.bucket_name`: replace the placeholder.

`templates`, `workflows`, `top_resources_count`, and `ai` are not written; add them when needed. `init` itself only checks that `account_mapping` is non-empty, so it runs fine against the placeholders.

## Validation

`scaffold`, `scaffold workflows`, and `validate` run these checks in order after loading the file. The first failure is returned, prefixed with `configuration validation failed:`.

| Check | Error |
| --- | --- |
| `provider.aws` section exists | `AWS provider configuration is required` |
| `account_mapping` has at least one entry | `AWS account mapping is required in provider configuration` |
| every account ID matches `^\d{12}$` | `AWS account ID must be a 12-digit number: Update the account mapping "dev": "REPLACE_WITH_YOUR_DEV_ACCOUNT_ID"` |
| `bucket_name` is set and not whitespace | `backend.s3.bucket_name is invalid: must not be empty` |
| `bucket_name` is not the seed placeholder | `backend.s3.bucket_name is invalid: placeholder value must be replaced with actual bucket name` |

Two more checks happen during rendering:

- `--env` must be a key in `account_mapping`. Otherwise: `no account mapping found for environment "prod", available: [dev, prd, stg]`.
- `workflows.name` must not contain `{{` or `}}`. Otherwise: `name_template produced invalid filename: name must be a plain string without Go template syntax (e.g. 'my-terraform'), got: ...`.

`init` with an existing file that lacks `account_mapping` fails with `.tfskel.yaml: provider.aws.account_mapping is missing or empty; add entries mapping environments to AWS account IDs`.

## Deprecated keys

Each produces a warning at load time and is otherwise ignored.

| Key | Replacement | Warning |
| --- | --- | --- |
| `generate` | `templates`, `workflows` | `DEPRECATION: 'generate' section is deprecated, use 'templates' and 'workflows' sections instead - this will be ignored` |
| `templates_dir` (root) | `templates.dir` | `DEPRECATION: 'templates_dir' at root level is deprecated, use 'templates.dir' instead (current: <value>) - this will be ignored` |
| `extra_template_extensions` | none; every `.tmpl` file is a template | `DEPRECATION: 'extra_template_extensions' is no longer supported - all .tmpl files are now processed as templates` |
| `top_n_count` | `top_resources_count` | `DEPRECATION: 'top_n_count' is no longer supported - use 'top_resources_count' instead` |

## Debugging config resolution

`--verbose` prints the file in use and, for `scaffold` and `validate`, one line per resolved key naming its source (`flag --<name>`, `config file <path>`, or `default`):

```text
Using config file: /path/to/project/.tfskel.yaml
2026-11-01T10:15:02 [DEBUG  ] logger.go:160 Config backend.s3.bucket_name = my-tfstate-{{.Env}} (from config file /path/to/project/.tfskel.yaml)
2026-11-01T10:15:02 [DEBUG  ] logger.go:160 Config templates.dir = ./templates (from flag --templates-dir)
```

The `logger.go:160` token is the logger's own call site, not the config code; ignore it.

Keys reported: `terraform_version`, `backend.s3.bucket_name`, `provider.aws.version`, `templates.dir`, `workflows.create`, `workflows.name`, `workflows.aws_role_name`, `workflows.aws_role_arn`. Unset keys are skipped. `TFSKEL_LOG_LEVEL=debug` has the same effect as `--verbose` for these lines.

## See also

- [`.tfskel.example.yaml`](../.tfskel.example.yaml): annotated copy of every key.
- [templates.md](templates.md): placeholders available in template-aware keys and custom templates.
- [`tfskel init`](reference/tfskel_init.md), [`tfskel scaffold`](reference/tfskel_scaffold.md), [`tfskel scaffold workflows`](reference/tfskel_scaffold_workflows.md), [`tfskel validate`](reference/tfskel_validate.md), [`tfskel review plan`](reference/tfskel_review_plan.md): flag reference.
