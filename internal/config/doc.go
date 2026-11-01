// Package config loads and validates .tfskel.yaml.
//
// Load unmarshals the viper instance the CLI prepared, warns about
// deprecated keys, applies the few flags that override config keys
// (--templates-dir, --s3-bucket-name, --workflows), fills defaults, and
// logs where each value came from when verbose. Validate then enforces the
// rules every generating command depends on: the AWS provider section and
// account mapping exist, every account ID is 12 digits, and the S3 bucket
// name is set and is not the init placeholder.
//
// The package knows nothing about templates or the filesystem. Values that
// accept Go template syntax (bucket name, workflow role ARN and name) are
// stored verbatim here and rendered by internal/generate at scaffold time.
// Keys consumed only by review plan (critical_resources, top_resources_count,
// ai.*) are read by internal/plan and internal/ai directly from viper, not
// through Config.
//
// Every key and its validation error is documented in docs/configuration.md.
package config
