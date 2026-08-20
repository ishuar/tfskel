package cmd

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

// Version is the current tfskel release version.
// This value is automatically updated by release-please during releases.
// https://github.com/googleapis/release-please/blob/main/docs/customizing.md#updating-arbitrary-files
const Version = "0.8.5" // x-release-please-version

const repoURL = "https://github.com/ishuar/tfskel"

// BuiltBy identifies the build pipeline that produced this binary.
// GoReleaser overrides this to "goreleaser" for official release builds;
// all other builds (make, go build, go install) leave it as "source".
// Used to decide whether to render a release URL — we only link to a release
// page when we're certain the binary came from that release pipeline.
//
//nolint:gochecknoglobals // set by ldflags at build time
var BuiltBy = "source"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Show tfskel version, build commit, and source/release URL.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), buildVersionInfo())
			return err
		},
	}
}

// buildVersionInfo returns the version string shared by `tfskel version` and
// the `--version` flag. Output differs by build origin.
func buildVersionInfo() string {
	if BuiltBy == "goreleaser" {
		// GoReleaser injects RFC3339 (e.g. "2026-04-17T00:03:27Z"); keep just the date.
		date := strings.SplitN(Date, "T", 2)[0]
		return fmt.Sprintf(
			"tfskel %s (%s)\ncommit:  %s\nos/arch: %s/%s\n%s/releases/tag/v%s\n",
			Version, date, Commit, runtime.GOOS, runtime.GOARCH, repoURL, Version,
		)
	}
	return fmt.Sprintf(
		"tfskel %s (local build)\ncommit:  %s\nos/arch: %s/%s\n",
		Version, Commit, runtime.GOOS, runtime.GOARCH,
	)
}
