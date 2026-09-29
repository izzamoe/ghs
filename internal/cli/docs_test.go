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

// readmeSection returns the README text after the exact heading line, up to
// the next heading of the same or a higher level outside fenced blocks.
func readmeSection(t *testing.T, heading string) string {
	t.Helper()
	level := strings.Index(heading, " ")
	var b strings.Builder
	found, inFence := false, false
	for line := range strings.SplitSeq(readRepoFile(t, "README.md"), "\n") {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		if !found {
			found = !inFence && line == heading
			continue
		}
		if n := len(line) - len(strings.TrimLeft(line, "#")); !inFence && n > 0 && n <= level && strings.HasPrefix(line[n:], " ") {
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if !found {
		t.Fatalf("README.md has no %q heading", heading)
	}
	return b.String()
}

func TestReadmeInstallUsesModulePath(t *testing.T) {
	t.Parallel()
	mod := regexp.MustCompile(`(?m)^module (\S+)$`).FindStringSubmatch(readRepoFile(t, "go.mod"))
	if mod == nil {
		t.Fatal("go.mod has no module line")
	}
	want := "go install " + mod[1] + "/cmd/ghs@latest"
	if !strings.Contains(readmeSection(t, "### Option A: Go toolchain"), want) {
		t.Fatalf("README Option A does not contain %q", want)
	}
}

// goreleaserArchives renders the archive names .goreleaser.yaml produces,
// with <version>, <os>, and <arch> placeholders, plus every os_arch pair.
func goreleaserArchives(t *testing.T) (patterns []string, pairs []string, checksums string) {
	t.Helper()
	cfg := readRepoFile(t, ".goreleaser.yaml")
	list := func(re string) []string {
		m := regexp.MustCompile(re).FindStringSubmatch(cfg)
		if m == nil {
			t.Fatalf(".goreleaser.yaml: no match for %s", re)
		}
		var items []string
		for item := range strings.SplitSeq(m[1], ",") {
			items = append(items, strings.TrimSpace(item))
		}
		return items
	}
	goos := list(`(?m)^\s+goos: \[(.*)\]`)
	goarch := list(`(?m)^\s+goarch: \[(.*)\]`)
	formats := list(`(?m)^    formats: \[(.*)\]`)
	project := regexp.MustCompile(`(?m)^project_name: (\S+)$`).FindStringSubmatch(cfg)
	archive := regexp.MustCompile(`(?m)^    name_template: "(.*)"$`).FindStringSubmatch(cfg)
	sums := regexp.MustCompile(`(?ms)^checksum:\s*\n\s+name_template: (\S+)$`).FindStringSubmatch(cfg)
	if project == nil || archive == nil || sums == nil {
		t.Fatal(".goreleaser.yaml: project_name, archive name_template, or checksum name_template not found")
	}
	windows := regexp.MustCompile(`(?m)- goos: windows\s*\n\s+formats: \[(.*)\]`).FindStringSubmatch(cfg)

	name := strings.NewReplacer(
		"{{ .ProjectName }}", project[1],
		"{{ .Version }}", "<version>",
		"{{ .Os }}", "<os>",
		"{{ .Arch }}", "<arch>",
	).Replace(archive[1])
	if strings.Contains(name, "{{") {
		t.Fatalf("archive name_template %q uses a field this test does not know", archive[1])
	}
	for _, f := range formats {
		patterns = append(patterns, name+"."+f)
	}
	if windows != nil {
		patterns = append(patterns, strings.Replace(name, "<os>", "windows", 1)+"."+strings.TrimSpace(windows[1]))
	}
	for _, o := range goos {
		for _, a := range goarch {
			pairs = append(pairs, o+"_"+a)
		}
	}
	return patterns, pairs, sums[1]
}

func TestReadmeReleaseArchivesMatchGoreleaser(t *testing.T) {
	t.Parallel()
	section := readmeSection(t, "### Option B: Release archive")
	patterns, pairs, checksums := goreleaserArchives(t)
	for _, want := range append(append(patterns, pairs...), checksums) {
		if !strings.Contains(section, want) {
			t.Errorf("README Option B does not mention %q (from .goreleaser.yaml)", want)
		}
	}
	for _, want := range []string{"sha256sum --check --ignore-missing", "shasum -a 256", "Get-FileHash"} {
		if !strings.Contains(section, want) {
			t.Errorf("README Option B does not give the checksum command %q", want)
		}
	}
	install := readmeSection(t, "## Install")
	if !strings.Contains(install, "ghs is not distributed through Homebrew, Scoop, winget, apt, or the AUR; the two options above are the only supported install paths.") {
		t.Error("README Install does not state that no package-manager distribution exists")
	}
}

// maintainerIdentity is the maintainer's real login and name; README
// examples use neutral placeholders instead (FR-018).
var maintainerIdentity = []string{"zamyb", "IZZAMUDDIN", "Izzam <", "Izzamuddin", "Royhul"}

func TestReadmeUsesNeutralExamples(t *testing.T) {
	t.Parallel()
	readme := readRepoFile(t, "README.md")
	for _, unwanted := range maintainerIdentity {
		if strings.Contains(readme, unwanted) {
			t.Errorf("README.md contains %q; use alice/alice-work/Alice Example", unwanted)
		}
	}
}

// readmeHeadings is the README outline, in order, from
// specs/002-public-discoverability/contracts/docs.md §2.1.
var readmeHeadings = []string{
	"# ghs",
	"## Contents",
	"## What ghs does",
	"## Install",
	"### Requirements",
	"### Option A: Go toolchain",
	"### Option B: Release archive",
	"### Verify the install",
	"### Do not use sudo",
	"### Updating",
	"### Uninstall",
	"## First run",
	"## Commands",
	"### Exit codes",
	"### Flag rules",
	"## Command reference",
	"## Profiles",
	"## Switching: `use`",
	"## Workspaces",
	"## Seeing where you are: `list`, `status`, `doctor`",
	"## Remotes and cloning",
	"## Removing a profile",
	"## Troubleshooting",
	"### Symptoms and fixes",
	"### What ghs changes and how to undo it",
	"## Config and data",
	"### Files ghs writes",
	"### Files ghs reads",
	"### Network access",
	"### What ghs never does",
	"### Config file format",
	"## Cross-platform notes",
	"### Linux",
	"### macOS",
	"### Windows",
	"## Contributing",
	"## Security",
	"## Support",
	"## License",
}

func TestReadmeSectionOrder(t *testing.T) {
	t.Parallel()
	lines := strings.Split(stripFences(readRepoFile(t, "README.md")), "\n")
	var level12 []string
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			level12 = append(level12, line)
		}
	}
	pos := 0
	for _, heading := range readmeHeadings {
		i := slices.Index(lines[pos:], heading)
		if i < 0 {
			t.Fatalf("README.md heading %q is missing or out of order (expected after line %d of the outline)", heading, pos)
		}
		pos += i + 1
	}
	var want []string
	for _, h := range readmeHeadings {
		if !strings.HasPrefix(h, "### ") {
			want = append(want, h)
		}
	}
	if !slices.Equal(level12, want) {
		t.Fatalf("README.md level-1/2 headings differ from the contract.\ngot:\n%s\nwant:\n%s", strings.Join(level12, "\n"), strings.Join(want, "\n"))
	}
	// Contents links every level-2 heading.
	contents := readmeSection(t, "## Contents")
	for _, h := range want[2:] {
		link := "](#" + headingAnchor(strings.TrimPrefix(h, "## ")) + ")"
		if !strings.Contains(contents, link) {
			t.Errorf("README Contents has no link %q for %q", link, h)
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

func TestReadmeCommandReferenceMatchesHelp(t *testing.T) {
	t.Parallel()
	section := readmeSection(t, "## Command reference")
	pos := 0
	for _, spec := range commands {
		heading := "### `ghs " + spec.Name + "`\n"
		block := "```\n" + spec.usageText() + "```\n"
		i := strings.Index(section[pos:], heading+"\n"+block)
		if i < 0 {
			t.Fatalf("README Command reference lacks, in command-table order, the block for %s. Expected:\n%s\n%s", spec.Name, heading, block)
		}
		pos += i + len(heading) + len(block)
	}
}

func TestReadmeMatchesHelpFooter(t *testing.T) {
	t.Parallel()
	help := helpOutput(t)
	config := regexp.MustCompile(`(?m)^Config: (\S+), or (\S+)$`).FindStringSubmatch(help)
	docs := regexp.MustCompile(`(?m)^Docs: (\S+)$`).FindStringSubmatch(help)
	if config == nil || docs == nil {
		t.Fatalf("help lacks the Config: or Docs: line:\n%s", help)
	}
	readme := readRepoFile(t, "README.md")
	for _, want := range []string{config[1], config[2], docs[1]} {
		if !strings.Contains(readme, want) {
			t.Errorf("README.md does not contain %q from ghs --help", want)
		}
	}
	data := readmeSection(t, "## Config and data")
	for _, want := range []string{config[1], config[2]} {
		if !strings.Contains(data, want) {
			t.Errorf("README Config and data does not contain %q", want)
		}
	}
}

// doctorCheckNames are the six doctor checks, in output order; the e2e
// doctor tests pin the same names.
var doctorCheckNames = []string{"gh", "git identity", "origin", "ssh config", "ssh auth", "workspace"}

func TestReadmeTroubleshootingCoversDoctorChecks(t *testing.T) {
	t.Parallel()
	section := readmeSection(t, "## Troubleshooting")
	var want []string
	for _, name := range doctorCheckNames {
		want = append(want, "`"+name+"`")
	}
	for _, outcome := range []string{outcomeOK, outcomeWarn, outcomeFail, outcomeSkip} {
		want = append(want, "`"+outcome+"`")
	}
	want = append(want,
		// symptoms (contracts/docs.md §2.5)
		"Permission denied (publickey)", "Host key verification failed", "restored origin to",
		"restored gh account", "could not restore", "did you mean", "has no email",
		"is not logged in for github.com", "is not a GitHub remote", "not available for local builds",
		"sudo", "OpenSSH Client",
		// manual undo commands (contracts/docs.md §2.6)
		"gh auth switch --user", "git remote set-url origin", "git config --global --unset --fixed-value",
		"gh ssh-key delete", "git commit --amend --reset-author", "ssh -T git@",
	)
	for _, w := range want {
		if !strings.Contains(section, w) {
			t.Errorf("README Troubleshooting does not mention %q", w)
		}
	}
	// Recovery never asks the user to delete keys with ghs or log out.
	for _, unwanted := range []string{"gh auth logout --hostname github.com --user"} {
		if strings.Contains(readmeSection(t, "### Symptoms and fixes"), unwanted) {
			t.Errorf("README Symptoms and fixes must not instruct %q", unwanted)
		}
	}
}
