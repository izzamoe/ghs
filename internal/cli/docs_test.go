package cli

import (
	"bytes"
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
