# Templates

tfskel generates every file from a Go `text/template`. The templates ship embedded in the binary; `tfskel scaffold` can also load a directory of your own `*.tmpl` files that add to or replace the embedded ones. Each template receives one `Data` struct, a small set of string helper functions, and gets a few marker comments injected at write time so later runs can detect what changed. This page documents all of that. Flags are listed in the command reference, not here.

## Embedded templates

Source tree: [`internal/templates/files/`](../internal/templates/files/). Files ending in `.tmpl` are parsed as Go templates. Everything else (the `github/*.yaml` reusable workflows) is copied byte for byte, because GitHub's `${{ }}` syntax collides with Go template delimiters.

| Template | Rendered by | Output path | Notes |
| --- | --- | --- | --- |
| `tf/backend.tf.tmpl` | [`tfskel scaffold`](reference/tfskel_scaffold.md) | `envs/<env>/<region>/<app>/backend.tf` | S3 backend block; carries `tfskel-metadata` |
| `tf/versions.tf.tmpl` | `tfskel scaffold` | `envs/<env>/<region>/<app>/versions.tf` | `required_version`, AWS provider, `default_tags`; carries `tfskel-metadata` and `tfskel-tags-hash` |
| `root/.pre-commit-config.yaml.tmpl` | [`tfskel init`](reference/tfskel_init.md) | `.pre-commit-config.yaml` | init-managed, upgradable |
| `root/.tflint.hcl.tmpl` | `tfskel init` | `.tflint.hcl` | init-managed, upgradable |
| `root/trivy.yaml.tmpl` | `tfskel init` | `trivy.yaml` | init-managed, upgradable |
| `root/.gitignore.tmpl` | `tfskel init` | `.gitignore` | user-owned: no marker, never touched again |
| `root/.mise.toml.tmpl` | `tfskel init` | `.mise.toml` | user-owned: no marker, never touched again |
| `root/.terraform-version.tmpl` | `tfskel init` | `envs/<env>/.terraform-version`, one per environment | bare version string, no marker possible |
| `github/lint.yaml`, `github/reusable-*.yaml` | `tfskel init --workflows` (or `workflows.create: true`) | `.github/workflows/<same name>` | static copies |
| `github/terraform-plan-apply.yaml.tmpl` | [`tfskel scaffold workflows`](reference/tfskel_scaffold_workflows.md) | `.github/workflows/<env>-terraform-plan-apply.yaml` | name changes with `workflows.name`, see below |

The output path is decided by the first path segment of the template name (`tf/`, `root/`, `github/`), not by anything inside the template. `tfskel init` always uses the embedded templates; it does not read `templates.dir`. `.tf`, `.tfvars` and `.hcl` files written by `scaffold` are passed through `hclwrite.Format` after rendering, so the result matches `terraform fmt` and a later `fmt` run is a no-op.

Put together, `tfskel init` followed by `tfskel scaffold myapp --env dev --region eu-central-1` and `tfskel scaffold workflows --env dev` produces:

```text
.tfskel.yaml
.gitignore
.mise.toml
.pre-commit-config.yaml
.tflint.hcl
trivy.yaml
.github/workflows/
    lint.yaml                      # only with init --workflows
    reusable-*.yaml                # only with init --workflows
    dev-terraform-plan-apply.yaml  # from scaffold workflows
envs/
    dev/
        .terraform-version
        eu-central-1/
            myapp/
                backend.tf
                versions.tf
    stg/
        .terraform-version
    prd/
        .terraform-version
```

## Custom templates

Point `templates.dir` in `.tfskel.yaml` (or `--templates-dir`, which overrides it) at a directory. Rules, from `internal/templates/renderer.go`:

- Every file in that directory tree whose name ends in `.tmpl` is parsed as a Go template. Files without `.tmpl` are ignored, not copied.
- The output file name is the template file name without `.tmpl`. Subdirectories inside `templates.dir` do not matter; only the base name is used.
- Every custom template is registered under `tf/<name>` and therefore lands in the app directory, `envs/<env>/<region>/<app>/`. Custom templates cannot produce root files or workflows.
- A custom file named like an embedded `tf/` template (`backend.tf.tmpl`, `versions.tf.tmpl`) replaces it. Any other name adds a new file.
- A missing directory is an error. A template that fails to parse is an error. A template that fails to execute (for example, referencing a field that does not exist) is logged and skipped; the rest of the run continues.
- Existing output files are never overwritten by a plain `scaffold` run. Use `--upgrade` to re-render them; see [upgrading.md](upgrading.md).

To emit a literal `{{` (GitHub Actions expressions, Helm, etc.) write `{{"{{"}}`.

## Template data

Every template receives a pointer to this struct (`templates.Data`). Which fields are filled depends on the command.

| Placeholder | Source | Example |
| --- | --- | --- |
| `.Env` | `--env` | `dev` |
| `.Region` | `--region` | `eu-central-1` |
| `.AppDir` | positional `<app-dir>` argument, spaces replaced by `-` | `myapp` |
| `.AccountID` | `provider.aws.account_mapping[<env>]`; missing entry is an error | `123456789012` |
| `.ShortRegion` | derived from `.Region`: split on `-`, keep numeric and 1-2 letter parts, else first letter | `eu-central-1` → `euc1`, `ap-southeast-1` → `aps1` |
| `.S3BucketName` | `backend.s3.bucket_name` or `--s3-bucket-name`, rendered as a template; default `CHANGE_ME_WITH_YOUR_GLOBALLY_UNIQUE_S3_BUCKET_NAME` | `tfstate-dev-123456789012` |
| `.TerraformVersion` | `terraform_version` | `~> 1.13` |
| `.AWSProviderVersion` | `provider.aws.version`; default `~> 6.0` | `~> 6.0` |
| `.DefaultTags` | `provider.aws.default_tags`, keys lowercased (`CostCenter` → `costcenter`) | `map[managed_by:terraform]` |
| `.AWSRoleArn` | `workflows.aws_role_arn`, else built from `workflows.aws_role_name`, else placeholder (see below) | `arn:aws:iam::123456789012:role/dev-githubactionsrole` |
| `.WorkflowFileName` | computed file name of the workflow being rendered; set only for `github/*.tmpl` | `dev-terraform-plan-apply.yaml` |
| `.Environments` | keys of `account_mapping` (or `dev`, `stg`, `prd` without a config); set only by `init` for `.mise.toml` | `[dev prd stg]` |

Per command:

- `tfskel scaffold` fills everything except `.WorkflowFileName` and `.Environments`.
- `tfskel scaffold workflows` fills only `.Env`, `.AccountID`, `.AWSRoleArn` and `.WorkflowFileName`. `.Region`, `.AppDir` and `.ShortRegion` are empty there, including inside `aws_role_arn` / `aws_role_name` templates.
- `tfskel init` renders `.mise.toml` with `.TerraformVersion` (constraint already stripped to a plain version) and `.Environments`, `.terraform-version` with `.TerraformVersion`, and the remaining root files with an empty struct.

`.AWSRoleArn` resolution order:

1. `workflows.aws_role_arn` set: rendered as a template and used as is.
2. `workflows.aws_role_name` set: rendered as a template, then `arn:aws:iam::<AccountID>:role/<name>`.
3. Neither: `arn:aws:iam::<AccountID>:role/REPLACE_WITH_ROLE_TO_ASSUME`.

## Template functions

All functions come from `funcMap` in `internal/templates/renderer.go` and are available in embedded, custom and config-value templates. Argument order is the Go order, so a piped value becomes the last argument: `{{ .Env | toUpper }}` works, but `{{ .AppDir | replace "-" "_" }}` does not. Pass arguments explicitly for anything with more than one parameter.

| Function | Example | Result |
| --- | --- | --- |
| `replace s old new` | `{{ replace .AppDir "-" "_" }}` | `my_app` |
| `toLower s` | `{{ .Env \| toLower }}` | `dev` |
| `toUpper s` | `{{ .Env \| toUpper }}` | `DEV` |
| `trimSpace s` | `{{ trimSpace "  x " }}` | `x` |
| `trimPrefix s prefix` | `{{ trimPrefix .Region "eu-" }}` | `central-1` |
| `trimSuffix s suffix` | `{{ trimSuffix .AppDir "-api" }}` | `my` |
| `hasPrefix s prefix` | `{{ if hasPrefix .Env "prd" }}...{{ end }}` | `true` / `false` |
| `hasSuffix s suffix` | `{{ if hasSuffix .Region "-1" }}...{{ end }}` | `true` / `false` |
| `contains s substr` | `{{ if contains .Region "central" }}...{{ end }}` | `true` / `false` |
| `join elems sep` | `{{ join .Environments ", " }}` | `dev, prd, stg` |
| `split s sep` | `{{ range split .Region "-" }}{{ . }} {{ end }}` | `eu central 1 ` |
| `stripConstraint v` | `{{ stripConstraint .TerraformVersion }}` | `~> 1.13.1` → `1.13.1` |
| `miseTools` | `{{ range miseTools }}{{ .Name }} = "{{ .Version }}"{{ end }}` | one line per tool |

`stripConstraint` deletes every occurrence of `~>`, `>=`, `<=`, `>`, `<` and `=` from the string and trims the result. It does not parse ranges: `>= 1.10, < 2.0` becomes `1.10, 2.0`.

`miseTools` returns a fixed, name-sorted list of `{Name, Version}` entries for `awscli`, `pre-commit`, `tflint` and `trivy`, all with version `latest`. Terraform is not in the list; `.mise.toml.tmpl` adds it from `.TerraformVersion`. Pin versions by editing the generated `.mise.toml`.

## Config values that are templates

Some `.tfskel.yaml` strings are themselves rendered with the same data and functions before they are placed into templates. A value without `{{` is used verbatim.

| Config key | Templated | Data available |
| --- | --- | --- |
| `backend.s3.bucket_name` | yes | full `scaffold` data |
| `workflows.aws_role_arn` | yes | `.Env`, `.Region`, `.AppDir`, `.AccountID`, `.ShortRegion` |
| `workflows.aws_role_name` | yes | same as `aws_role_arn` |
| `workflows.name` | no | `{{` or `}}` in the value is an error |

`workflows.name` is a plain file-name stem. A trailing `.yaml` is stripped, then the file becomes `<env>-<name>.yaml`; without it the file is `<env>-terraform-plan-apply.yaml`. Names containing `/`, `\` or `..` are rejected.

```yaml
backend:
  s3:
    bucket_name: "tfstate-{{.Env}}-{{.AccountID}}-{{.ShortRegion}}"
workflows:
  create: true
  name: "terraform"              # -> dev-terraform.yaml
  aws_role_name: "{{.Env}}-githubactionsrole"
```

## Markers in generated files

tfskel does not put tracking comments in the template text. It injects them when it writes the file, using the comment syntax of the output extension (`##` for `.tf`/`.hcl`, `<!-- -->` for `.md`, `#` otherwise).

```hcl
## tfskel-source: {"template":"tf/versions.tf.tmpl","hash":"3f2a9c1e8b7d6a50"}
## tfskel-metadata: {"aws_provider_ver":"~> 6.0","tf_ver":"~> 1.13"}
## tfskel-tags-hash: 9c4e1b2a7d3f8e60
```

- `tfskel-source` (line 1 of every rendered file except `.terraform-version`, `.gitignore` and `.mise.toml`) records the template name and a hash of the template text. `--upgrade` uses it to decide what may be re-rendered; see [upgrading.md](upgrading.md).
- `tfskel-metadata` records the config values the file was rendered from. `backend.tf` stores `{"bucket": ...}`; `versions.tf` stores `{"aws_provider_ver": ..., "tf_ver": ...}`.
- `tfskel-tags-hash` (`versions.tf` only, omitted when `default_tags` is empty) is the first 16 hex characters of the SHA-256 of the lowercased tag map as JSON.

On every `tfskel scaffold <app>` run against an app directory that already exists, before any template is processed, tfskel reads `backend.tf` and `versions.tf`, compares their markers with the current config, and re-renders the whole file from its template when the bucket name, Terraform version, provider version or tags differ. A file with no `tfskel-metadata` line is treated as changed and re-rendered too. This happens without `--upgrade`. The file is replaced, not patched: hand edits to `backend.tf` or `versions.tf` are lost on such a rewrite. Output such as `Updated versions.tf - tf_ver changed: ~> 1.8 -> ~> 1.13` tells you what triggered it.

Marker injection is keyed on the template name, not on the file content. A custom `backend.tf.tmpl` or `versions.tf.tmpl` therefore gets the same markers and the same change detection automatically. Do not write `tfskel-metadata` or `tfskel-tags-hash` lines into a template yourself; tfskel adds them. Custom templates with any other name receive only the `tfskel-source` line.

## Example custom template

`templates/locals.tf.tmpl`:

```hcl
## Naming helpers for {{ .AppDir }} in {{ .Env }}/{{ .Region }}

locals {
  name_prefix = "{{ .AppDir }}-{{ .Env }}-{{ .ShortRegion }}"
  account_id  = "{{ .AccountID }}"
{{- if hasPrefix .Env "prd" }}
  retention_days = 90
{{- else }}
  retention_days = 7
{{- end }}

  tags = {
{{- range $k, $v := .DefaultTags }}
    {{ $k }} = "{{ $v }}"
{{- end }}
  }
}
```

```bash
tfskel scaffold myapp --env dev --region eu-central-1 --templates-dir ./templates
```

Produces `envs/dev/eu-central-1/myapp/locals.tf` with a `tfskel-source` line on top, formatted like `terraform fmt`, next to the embedded `backend.tf` and `versions.tf`.
