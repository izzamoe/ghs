package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

func helpOutput(t *testing.T) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if err := New(&out, &errOut).Run([]string{"--help"}); err != nil {
		t.Fatalf("help: %v", err)
	}
	return out.String()
}

func TestHelpStatesGitHubOnly(t *testing.T) {
	t.Parallel()
	help := helpOutput(t)
	for _, want := range []string{"Only github.com is supported", "Do not run ghs with sudo"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help does not contain %q:\n%s", want, help)
		}
	}
}

func readRepoFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return strings.ReplaceAll(string(data), "\r\n", "\n")
}

func TestReadmeGoVersionMatchesGoMod(t *testing.T) {
	t.Parallel()
	mod := regexp.MustCompile(`(?m)^go (\d+\.\d+(?:\.\d+)?)$`).FindStringSubmatch(readRepoFile(t, "go.mod"))
	if mod == nil {
		t.Fatal("go.mod has no go directive")
	}
	readme := regexp.MustCompile(`Requires Go (\d+\.\d+(?:\.\d+)?)`).FindAllStringSubmatch(readRepoFile(t, "README.md"), -1)
	if len(readme) == 0 {
		t.Fatal(`README.md has no "Requires Go <version>" line`)
	}
	for _, m := range readme {
		if m[1] != mod[1] {
			t.Fatalf("README says Go %s, go.mod says %s", m[1], mod[1])
		}
	}
}

// readmeCommandBlock returns the lines of the first fenced block after the
// "## Commands" heading.
func readmeCommandBlock(t *testing.T) []string {
	t.Helper()
	readme := readRepoFile(t, "README.md")
	_, after, ok := strings.Cut(readme, "\n## Commands\n")
	if !ok {
		t.Fatal(`README.md has no "## Commands" section`)
	}
	_, block, ok := strings.Cut(after, "```\n")
	if !ok {
		t.Fatal("no fenced block in the Commands section")
	}
	block, _, _ = strings.Cut(block, "```")
	return strings.Split(strings.TrimRight(block, "\n"), "\n")
}

func TestReadmeCommandListMatchesHelp(t *testing.T) {
	t.Parallel()
	readme := readmeCommandBlock(t)
	want := commandListLines()
	if !slices.Equal(readme, want) {
		t.Fatalf("README command block differs from help.\nREADME:\n%s\nhelp:\n%s", strings.Join(readme, "\n"), strings.Join(want, "\n"))
	}
	help := helpOutput(t)
	for _, line := range want {
		if !strings.Contains(help, "  "+line+"\n") {
			t.Fatalf("help does not list %q", line)
		}
	}
}

func TestReadmeStatesExitCodesGitHubOnlyAndNoSudo(t *testing.T) {
	t.Parallel()
	readme := readRepoFile(t, "README.md")
	for _, want := range []string{"exit code", "Only github.com is supported", "sudo", "`0`", "`1`", "`2`", "make check", "LICENSE"} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README.md does not mention %q", want)
		}
	}
	license := readRepoFile(t, "LICENSE")
	for _, want := range []string{"MIT License", "Copyright (c) 2026 izzamoe", "WITHOUT WARRANTY OF ANY KIND"} {
		if !strings.Contains(license, want) {
			t.Fatalf("LICENSE does not contain %q", want)
		}
	}
	changelog := readRepoFile(t, "CHANGELOG.md")
	for _, want := range []string{"Keep a Changelog", "## [Unreleased]", "**Breaking:**", "`ghs remove", "`ghs doctor", "`ghs workspace", "--fix-remote", "`GH AUTH`", "mismatch:"} {
		if !strings.Contains(changelog, want) {
			t.Fatalf("CHANGELOG.md does not contain %q", want)
		}
	}
}

func TestHelpListsEveryRegisteredCommand(t *testing.T) {
	t.Parallel()
	help := helpOutput(t)
	for _, spec := range commands {
		if !strings.Contains(help, "ghs "+spec.Name) {
			t.Fatalf("help does not list %s", spec.Name)
		}
		var out, errOut bytes.Buffer
		if err := New(&out, &errOut).Run([]string{spec.Name, "--help"}); err != nil {
			t.Fatalf("%s --help: %v", spec.Name, err)
		}
		for _, f := range spec.Flags {
			if !strings.Contains(out.String(), "--"+f.Name) {
				t.Fatalf("%s --help does not list --%s", spec.Name, f.Name)
			}
		}
	}
}

func TestHelpCommandAliasMatchesFlag(t *testing.T) {
	t.Parallel()
	for _, spec := range commands {
		var viaHelp, viaFlag, errOut bytes.Buffer
		if err := New(&viaHelp, &errOut).Run([]string{"help", spec.Name}); err != nil {
			t.Fatalf("help %s: %v", spec.Name, err)
		}
		if err := New(&viaFlag, &errOut).Run([]string{spec.Name, "--help"}); err != nil {
			t.Fatalf("%s --help: %v", spec.Name, err)
		}
		if viaHelp.String() != viaFlag.String() {
			t.Fatalf("ghs help %s differs from ghs %s --help:\n%s\n---\n%s", spec.Name, spec.Name, viaHelp.String(), viaFlag.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("help %s wrote to stderr: %q", spec.Name, errOut.String())
		}
	}
	// "help help", "help --help", and "help -h" ask for the general help.
	general := helpOutput(t)
	for _, topic := range []string{"help", "--help", "-h"} {
		var out, errOut bytes.Buffer
		if err := New(&out, &errOut).Run([]string{"help", topic}); err != nil || out.String() != general {
			t.Fatalf("help %s = %q, %v; want the general help", topic, out.String(), err)
		}
	}
}

func TestHelpUnknownTopicIsUsageError(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	err := New(&out, &errOut).Run([]string{"help", "nosuch"})
	var usageErr *UsageError
	if !errors.As(err, &usageErr) || !strings.Contains(usageErr.Message, `unknown command "nosuch"`) {
		t.Fatalf("help nosuch = %v, want the unknown-command usage error", err)
	}
	if out.Len() != 0 {
		t.Fatalf("help nosuch wrote to stdout: %q", out.String())
	}
}

func TestHelpSurplusArgumentIsUsageError(t *testing.T) {
	t.Parallel()
	var out, errOut bytes.Buffer
	err := New(&out, &errOut).Run([]string{"help", "use", "--global"})
	var usageErr *UsageError
	if !errors.As(err, &usageErr) || !strings.Contains(usageErr.Message, `unexpected argument "--global"`) {
		t.Fatalf("help use --global = %v, want a usage error", err)
	}
	if usageErr.Usage != generalUsage() {
		t.Fatalf("usage block = %q, want the general usage", usageErr.Usage)
	}
	if out.Len() != 0 {
		t.Fatalf("help use --global wrote to stdout: %q", out.String())
	}
}

const docsURL = "https://github.com/izzamoe/ghs#readme"

func TestHelpFooterHasDocsLine(t *testing.T) {
	t.Parallel()
	if help := helpOutput(t); !strings.HasSuffix(help, "Exit codes: 0 success, 1 failure, 2 usage error.\nDocs: "+docsURL+"\n") {
		t.Fatalf("help does not end with the Docs line:\n%s", help)
	}
}
