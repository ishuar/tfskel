package generate

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ishuar/tfskel/internal/config"
	"github.com/ishuar/tfskel/internal/fs"
	"github.com/ishuar/tfskel/internal/logger"
	"github.com/ishuar/tfskel/internal/templates"
)

func TestIsHCLOutput(t *testing.T) {
	cases := map[string]bool{
		"backend.tf":         true,
		"vars.tfvars":        true,
		"config.hcl":         true,
		"versions.TF":        true, // case-insensitive
		"README.md":          false,
		"workflow.yaml":      false,
		".gitignore":         false,
		".terraform-version": false,
	}
	for path, want := range cases {
		assert.Equal(t, want, isHCLOutput(path), path)
	}
}

func TestFormatIfHCL_Idempotent(t *testing.T) {
	in := "terraform {\n  required_version = \"~> 1.14\"\n  required_providers {\n    aws = {\n      source = \"hashicorp/aws\"\n      version = \"~> 6.0\"\n    }\n  }\n}\n"
	once := formatIfHCL("versions.tf", in)
	twice := formatIfHCL("versions.tf", once)
	assert.Equal(t, once, twice, "hclwrite.Format must be idempotent")
}

func TestFormatIfHCL_PreservesMarkerComments(t *testing.T) {
	in := "## tfskel-source: {\"template\":\"tf/versions.tf.tmpl\",\"hash\":\"abc\"}\n## tfskel-metadata: {\"tf_ver\":\"~> 1.14\"}\nterraform {\n  required_version = \"~> 1.14\"\n}\n"
	out := formatIfHCL("versions.tf", in)
	assert.Contains(t, out, "tfskel-source")
	assert.Contains(t, out, "tfskel-metadata")
}

// TestUpgrade_NoDriftAfterTerraformFmt is the regression test for the bug
// where every `--upgrade` after `terraform fmt` flagged versions.tf as
// drifted because the template emitted unaligned `=` while tf fmt aligned them.
func TestUpgrade_NoDriftAfterTerraformFmt(t *testing.T) {
	filesystem := fs.NewMemoryFileSystem()
	log := logger.New(false)
	cfg := &config.Config{
		TerraformVersion: "~> 1.14.5",
		Provider: &config.Provider{
			AWS: &config.AWSProvider{
				Version:     "~> 6.0",
				DefaultTags: map[string]string{"managed_by": "terraform"},
			},
		},
	}
	gen := NewGenerator(cfg, filesystem, log)
	gen.SetUpgrade(true, false)

	renderer, err := templates.NewRenderer()
	require.NoError(t, err)
	gen.renderer = renderer

	tmplName := "tf/versions.tf.tmpl"
	outputPath := "envs/dev/eu-central-1/test5/versions.tf"
	require.NoError(t, filesystem.MkdirAll("envs/dev/eu-central-1/test5", 0755))

	data := &templates.Data{
		Env:                "dev",
		Region:             "eu-central-1",
		AppDir:             "test5",
		ShortRegion:        "euc1",
		TerraformVersion:   "~> 1.14.5",
		AWSProviderVersion: "~> 6.0",
		DefaultTags:        map[string]string{"managed_by": "terraform"},
	}

	require.NoError(t, gen.writeTemplateFile(tmplName, outputPath, data))
	written, err := filesystem.ReadFile(outputPath)
	require.NoError(t, err)

	// The writer must produce already-formatted output: a second hclwrite.Format
	// pass (what `terraform fmt` does) is a no-op.
	reformatted := formatIfHCL(outputPath, string(written))
	assert.Equal(t, string(written), reformatted, "writer output should be terraform-fmt idempotent")

	// Simulate `terraform fmt` having run on a file written by an older tfskel
	// version that did not pre-format. The unaligned variant still has valid
	// source/metadata markers and a current template hash.
	unaligned := strings.ReplaceAll(string(written), "      env        = ", "      env = ")
	unaligned = strings.ReplaceAll(unaligned, "      app        = ", "      app = ")
	require.NoError(t, filesystem.WriteFile(outputPath, []byte(unaligned), 0644))

	require.NoError(t, gen.upgradeFileIfEligible(tmplName, outputPath, data))

	// Upgrade must treat the legacy unaligned file as up-to-date because the
	// only difference is whitespace alignment that hclwrite.Format normalizes.
	after, err := filesystem.ReadFile(outputPath)
	require.NoError(t, err)
	assert.Equal(t, unaligned, string(after), "upgrade should not rewrite a file that only differs by tf-fmt alignment")
}
