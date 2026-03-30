package toolcheck

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	symbolInstalled     = "✓"
	symbolMissing       = "✗"
	symbolMiseManaged   = "↓"
	symbolMiseInstalled = "!"
	symbolGlobalPath    = "~"
)

// FormatReport returns a human-readable status report for the tool check results.
func FormatReport(report *Report) string {
	var b strings.Builder

	b.WriteString("Running pre-flight checks...\n\n")

	// Separate global PATH tools from the rest for grouped display.
	var mainTools, globalTools []ToolResult
	for _, t := range report.Tools {
		if t.Status == StatusGlobalPath {
			globalTools = append(globalTools, t)
		} else {
			mainTools = append(mainTools, t)
		}
	}

	// Find max name length across all tools for consistent alignment.
	maxLen := 0
	for _, t := range report.Tools {
		if len(t.Name) > maxLen {
			maxLen = len(t.Name)
		}
	}

	for _, t := range mainTools {
		symbol, detail := formatToolLine(t)
		padding := strings.Repeat(" ", maxLen-len(t.Name)+2)
		if detail != "" {
			fmt.Fprintf(&b, "  %s %s%s%s\n", symbol, t.Name, padding, detail)
		} else {
			fmt.Fprintf(&b, "  %s %s\n", symbol, t.Name)
		}
	}

	if len(globalTools) > 0 {
		b.WriteString("\n  Installed globally (mise install will add .mise.toml version):\n")
		for _, t := range globalTools {
			padding := strings.Repeat(" ", maxLen-len(t.Name)+2)
			if t.Version != "" {
				fmt.Fprintf(&b, "  %s %s%sv%s\n", symbolGlobalPath, t.Name, padding, t.Version)
			} else {
				fmt.Fprintf(&b, "  %s %s\n", symbolGlobalPath, t.Name)
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(formatMiseStatus(report))

	return b.String()
}

// formatToolLine returns the status symbol and detail text for a single tool.
func formatToolLine(t ToolResult) (string, string) {
	switch t.Status {
	case StatusInstalled:
		if t.Version != "" {
			return symbolInstalled, "v" + t.Version
		}
		return symbolInstalled, "(installed)"
	case StatusMissing:
		return symbolMissing, ""
	case StatusMiseManaged:
		return symbolMiseManaged, "(will be installed by mise)"
	case StatusMiseInstalled:
		return symbolMiseInstalled, "(installed by mise, but not activated)"
	case StatusGlobalPath:
		return symbolGlobalPath, ""
	}
	return symbolMissing, ""
}

// formatMiseStatus returns the mise status line with actionable instructions.
func formatMiseStatus(report *Report) string {
	if !report.MiseFound {
		return fmt.Sprintf(`  mise: %s not found
  Install mise first:
    macOS:          brew install mise
    Ubuntu/Debian:  sudo apt install -y mise
    Fedora/RHEL:    sudo dnf install -y mise
    Any platform:   curl https://mise.run | sh
  Then run: mise install
`, symbolMissing)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "  mise: %s installed\n", symbolInstalled)

	hasMiseManaged := false
	hasMiseInstalled := false
	hasGlobalPath := false
	for _, t := range report.Tools {
		if t.Status == StatusMiseManaged {
			hasMiseManaged = true
		}
		if t.Status == StatusMiseInstalled {
			hasMiseInstalled = true
		}
		if t.Status == StatusGlobalPath {
			hasGlobalPath = true
		}
	}

	if hasMiseManaged || hasGlobalPath {
		b.WriteString("  Run: mise install\n")
	}
	if hasMiseInstalled {
		if report.MiseActivated {
			b.WriteString("  Tools installed by mise but not on PATH:\n")
			b.WriteString("  Run: mise reshim && exec $SHELL\n")
		} else {
			b.WriteString("  Activate mise in your shell:\n")
			b.WriteString(miseActivationHint(report.Shell))
			b.WriteString("  Then restart your shell: exec $SHELL\n")
		}
	}
	if !hasMiseManaged && !hasMiseInstalled && !hasGlobalPath {
		b.WriteString("  All tools are installed ✓\n")
	}

	return b.String()
}

// shellRCFile returns the rc file path for a given shell name.
func shellRCFile(shell string) string {
	home := "~"
	switch shell {
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return home + "/." + shell + "rc"
	}
}

// miseActivationHint returns shell-specific mise activation instructions.
// If shell is empty (undetected), it shows instructions for all common shells.
func miseActivationHint(shell string) string {
	switch shell {
	case "fish":
		rcFile := shellRCFile("fish")
		return fmt.Sprintf("  echo 'mise activate fish | source' >> %s\n", rcFile)
	case "bash", "zsh":
		rcFile := shellRCFile(shell)
		return fmt.Sprintf("  echo 'eval \"$(mise activate %s)\"' >> %s\n", shell, rcFile)
	default:
		// Unknown or empty shell — show all common options
		return "  eval \"$(mise activate bash)\"   # add to ~/.bashrc\n" +
			"  eval \"$(mise activate zsh)\"    # add to ~/.zshrc\n" +
			"  mise activate fish | source     # add to ~/.config/fish/config.fish\n"
	}
}
