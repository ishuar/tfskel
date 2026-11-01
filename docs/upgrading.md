# Upgrading generated files

tfskel never overwrites an existing file by default. When a template or `.tfskel.yaml` changes, pass `--upgrade` to re-render the files tfskel generated earlier. tfskel decides what to rewrite from a source marker embedded in each generated file, and only touches files whose re-rendered content differs from what is on disk. `--dry-run` previews every decision without writing.

## Source markers

Each generated file starts with a comment that records which template produced it and a hash of that template:

```hcl
## tfskel-source: {"template":"tf/backend.tf.tmpl","hash":"a1b2c3d4e5f60718"}
```

- `template` is the full template key, including the category prefix (`tf/`, `root/`, `github/`).
- `hash` is the first 16 hex characters of the SHA-256 of the raw template file as it was at generation time. It covers the template source, not the rendered output, so a config change alone does not change the hash.

The comment syntax follows the output file's extension (the `.tmpl` suffix is stripped first):

| Extension          | Marker line                                   |
| ------------------ | --------------------------------------------- |
| `.tf`, `.hcl`      | `## tfskel-source: {...}`                     |
| `.md`, `.markdown` | `<!-- tfskel-source: {...} -->`               |
| anything else      | `# tfskel-source: {...}` (YAML files)         |

Files without a marker:

- `envs/<env>/.terraform-version` holds a bare version string; a comment line would break it. It is still created and upgraded by `tfskel init`, but only via `--force` once it exists.
- `.gitignore` and `.mise.toml` are user-owned after creation. `tfskel init` never rewrites them, even with `--upgrade --force`.
- `.tfskel.yaml` is created once and never touched again.

`backend.tf` and `versions.tf` carry two extra comment lines after the source marker, `tfskel-metadata` and `tfskel-tags-hash`. `tfskel scaffold` uses these to update bucket name, versions and default tags in place on every run, with or without `--upgrade`. See [templates.md](templates.md) for details.

## Default behavior (no `--upgrade`)

Existing files are left alone and logged as `<path> already exists, skipping`. Missing files are created. The only exception is the automatic metadata update of `backend.tf` and `versions.tf` described above.

## `--upgrade`

For every output file that already exists, tfskel runs this decision. Steps for `tfskel scaffold` and `tfskel scaffold workflows` are in `internal/generate/upgrade.go`; `tfskel init` uses the same rules in `internal/bootstrap/runner.go`.

| Condition                                                        | Result    | Log line                                                             |
| ---------------------------------------------------------------- | --------- | -------------------------------------------------------------------- |
| Template not in `templates.upgrade` whitelist (scaffold only)   | skip      | debug only                                                           |
| No source marker, no `--force`                                   | skip      | `<path> has no source marker, skipping upgrade (use --force to override)` |
| No source marker, `--force`                                      | overwrite | `Force upgrading <path> (--force, no source marker)`                 |
| Marker present but malformed JSON, no `--force`                  | error     | `invalid source marker in <path>: ...`                               |
| Marker present but malformed JSON, `--force`                     | overwrite | `Force upgrading <path> (--force, invalid source marker: ...)`       |
| Marker names a different template                                | skip      | debug only                                                           |
| Re-rendered content equals the file on disk                      | skip      | debug only                                                           |
| Content differs and marker hash differs from current template    | overwrite | `Upgrading <path> (template: <old> -> <new>)`                        |
| Content differs but marker hash matches current template         | overwrite | `Upgrading <path> (content drift detected)`                          |

The last two rows mean `--upgrade` compares full rendered output, not just the hash. A changed template always rewrites, because the new hash lands in the marker line. An unchanged template whose output now differs, because a value in `.tfskel.yaml` moved or someone hand-edited the file, is rewritten too. For `.tf`, `.tfvars` and `.hcl` outputs both sides are normalized with `terraform fmt` rules first, so a prior `terraform fmt` run does not count as drift. `tfskel init` reports the same case as `config drift detected`.

Because unchanged files are skipped, `--upgrade` is safe to run repeatedly.

## `--force`

`--force` only changes what happens to files that have no marker or a broken one. It is rejected on its own:

```bash
tfskel scaffold myapp --env dev --region us-east-1 --force
# Error: --force can only be used together with --upgrade or --upgrade-all

tfskel init --force
# Error: --force can only be used together with --upgrade
```

Use it for files generated before source markers existed, or where the marker was removed by hand.

## Restricting with `templates.upgrade`

By default every template is eligible. To limit `--upgrade` to a subset, list template basenames in `.tfskel.yaml`:

```yaml
templates:
  upgrade:
    - backend.tf.tmpl
    - versions.tf.tmpl
```

Names are matched against the basename of the template key (`tf/backend.tf.tmpl` matches `backend.tf.tmpl`). Files from unlisted templates are skipped even if they carry a marker. The whitelist applies to `tfskel scaffold` and `tfskel scaffold workflows`; `tfskel init` does not read it.

## Command support

| Command                    | Flags                                         | Scope                                                                                             |
| -------------------------- | --------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `tfskel init`              | `--upgrade`, `--force`, `--skip`              | `.pre-commit-config.yaml`, `.tflint.hcl`, `trivy.yaml`, `envs/<env>/.terraform-version`, and the shared workflow files when workflows are enabled |
| `tfskel scaffold <app>`    | `--upgrade`, `--force`                        | `tf/` templates in `envs/<env>/<region>/<app>/`                                                   |
| `tfskel scaffold`          | `--upgrade-all`, `--skip`, `--force`          | Every non-hidden directory under `envs/<env>/<region>/`                                           |
| `tfskel scaffold workflows`| `--upgrade`, `--force`                        | `.github/workflows/<env>-terraform-plan-apply.yaml` (per-env `github/*.tmpl` templates)           |

Full flag lists: [`tfskel init`](reference/tfskel_init.md), [`tfskel scaffold`](reference/tfskel_scaffold.md), [`tfskel scaffold workflows`](reference/tfskel_scaffold_workflows.md).

`--skip` means different things per command:

- `tfskel init --upgrade --skip .tflint.hcl,trivy.yaml` skips the named files during upgrade. Requires `--upgrade`. A listed file that does not exist yet is still created.
- `tfskel scaffold --upgrade-all --skip base-infra,legacy` skips the named app directories. Requires `--upgrade-all`.

`--upgrade-all` takes no `<app-dir>` argument and cannot be combined with `--upgrade`. It fails if `envs/<env>/<region>/` does not exist or contains no app directories, and prints one summary for the whole batch.

## Dry run and the summary line

`--dry-run` is a global flag. It runs the full decision above against the real files but writes nothing. Log lines switch to `[dry-run] Would create`, `[dry-run] Would upgrade` and `[dry-run] Would force upgrade`, and the run ends with `Dry run complete — no files were written`.

`tfskel scaffold` and `tfskel scaffold workflows` print a one-line count of what happened:

```text
Summary: 1 file created, 3 files skipped, 2 files upgraded, 1 file force-upgraded
```

Under `--dry-run` the verbs become `would be created`, `would be upgraded` and `would be force-upgraded`. Categories with a zero count are omitted. `tfskel init` does not print a summary line.

## Typical workflow

1. Change the input. Either bump a value in `.tfskel.yaml` (`terraform_version`, `provider.aws.version`, `backend.s3.bucket_name`, `default_tags`) or edit a template in your `templates.dir`.
2. Preview:

```bash
tfskel scaffold myapp --env dev --region us-east-1 --upgrade --dry-run
```

3. Read the `Upgrading ...` and `has no source marker` lines. Add `--force` only if the unmarked files are ones you want rewritten.
4. Apply:

```bash
tfskel scaffold myapp --env dev --region us-east-1 --upgrade
```

5. For a whole region, or after upgrading the tfskel binary itself:

```bash
tfskel scaffold --upgrade-all --env prd --region eu-central-1 --dry-run
tfskel scaffold --upgrade-all --env prd --region eu-central-1
tfskel scaffold workflows --env prd --upgrade
tfskel init --upgrade
```

Review the diff with `git diff` before committing. Files that drifted because of a deliberate manual edit will be reverted to the template output; move such edits into a custom template instead.
