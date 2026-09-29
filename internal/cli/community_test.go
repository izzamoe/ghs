package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// emailAddress matches anything that looks like an email address (the
// domain ends in a letters-only label, so module@v1.2.3 is not one); the
// community files must not name one (no such contact exists).
var emailAddress = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}\b`)

// emailsIn returns the email-like strings in content, ignoring SSH remotes
// such as git@github.com, which are not contacts.
func emailsIn(content string) []string {
	var emails []string
	for _, m := range emailAddress.FindAllString(content, -1) {
		if !strings.HasPrefix(m, "git@") {
			emails = append(emails, m)
		}
	}
	return emails
}

// communityFiles is the required content of each community health file,
// from specs/002-public-discoverability/contracts/docs.md §3.
var communityFiles = []struct {
	path    string
	require []string
	forbid  []string
	noEmail bool
	// onlyURLs, when set, is the complete list of absolute URLs allowed.
	onlyURLs []string
}{
	{
		path: "CONTRIBUTING.md",
		require: []string{
			"make check", "gofmt -l", "go vet ./...", "go test -race", "make e2e", "internal/e2e",
			"hermetic", "failing test", "specs/", "SPECIFY_FEATURE_DIRECTORY=", "CHANGELOG.md",
			"## Maintainer runbook", "gh repo edit", "private-vulnerability-reporting", "gh release edit",
			"gh release view", "--json body", "maintainer-only",
		},
		forbid:  []string{"brew install", "scoop install", "winget install"},
		noEmail: true,
	},
	{
		path: "SECURITY.md",
		require: []string{
			"## Supported versions", "latest release", "## Reporting a vulnerability",
			"https://github.com/izzamoe/ghs/security/advisories/new", "security report, please contact me",
			"## Scope", "private key", "## Out of scope", "no bug bounty",
		},
		noEmail: true,
	},
	{
		path: "SUPPORT.md",
		require: []string{
			"ghs doctor --offline", "ghs status", "README.md#troubleshooting", "search",
			"best effort", "one maintainer", "no response-time",
		},
		noEmail: true,
	},
	{
		path: "CODE_OF_CONDUCT.md",
		require: []string{
			"Contributor Covenant", "version 2.1", "@izzamoe", "security/advisories/new",
			"https://www.contributor-covenant.org",
		},
		forbid:  []string{"[INSERT CONTACT METHOD]"},
		noEmail: true,
	},
	{
		path: ".github/ISSUE_TEMPLATE/bug_report.yml",
		require: []string{
			"name: Bug report", "id: version", "id: platform", "id: command", "id: expected", "id: actual",
			"id: doctor", "ghs doctor --offline", "Do not paste private keys",
		},
		noEmail: true,
	},
	{
		path:    ".github/ISSUE_TEMPLATE/feature_request.yml",
		require: []string{"name: Feature request", "id: problem", "id: proposal", "id: alternatives"},
		noEmail: true,
	},
	{
		path:    ".github/ISSUE_TEMPLATE/config.yml",
		require: []string{"blank_issues_enabled: false", "SUPPORT.md", "README.md#troubleshooting"},
		noEmail: true,
	},
	{
		path:    ".github/PULL_REQUEST_TEMPLATE.md",
		require: []string{"make check", "failing test", "CHANGELOG.md", "README.md", "--help", "no new runtime dependency"},
		noEmail: true,
	},
	{
		path:    ".github/dependabot.yml",
		require: []string{`package-ecosystem: "github-actions"`, `interval: "weekly"`},
	},
	{
		path:     "docs/assets/social-preview.svg",
		require:  []string{`viewBox="0 0 1280 640"`, "ghs", "github.com/izzamoe/ghs"},
		forbid:   []string{"<image", "@import", `href="http`, "<script"},
		onlyURLs: []string{"http://www.w3.org/2000/svg"},
	},
}

func TestCommunityFilesPresentAndComplete(t *testing.T) {
	t.Parallel()
	for _, f := range communityFiles {
		t.Run(f.path, func(t *testing.T) {
			t.Parallel()
			content := readRepoFile(t, f.path)
			for _, want := range f.require {
				if !strings.Contains(content, want) {
					t.Errorf("%s does not contain %q", f.path, want)
				}
			}
			for _, unwanted := range f.forbid {
				if strings.Contains(content, unwanted) {
					t.Errorf("%s must not contain %q", f.path, unwanted)
				}
			}
			if f.onlyURLs != nil {
				for _, u := range regexp.MustCompile(`https?://[^\s"'<>]+`).FindAllString(content, -1) {
					if !slices.Contains(f.onlyURLs, u) {
						t.Errorf("%s references %q; only %v are allowed", f.path, u, f.onlyURLs)
					}
				}
			}
			if f.noEmail {
				for _, m := range emailsIn(content) {
					t.Errorf("%s names an email address %q; no such contact exists", f.path, m)
				}
			}
		})
	}
	// No funding account exists, so no funding file may exist either.
	for _, name := range []string{".github/FUNDING.yml", "FUNDING.yml", "docs/FUNDING.yml"} {
		if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(name))); err == nil {
			t.Errorf("%s exists, but the project has no funding account", name)
		}
	}
}

// docMakeTargets returns every `make <target>` the document mentions, inline
// in backticks or as a command line inside a fenced block.
func docMakeTargets(content string) []string {
	var targets []string
	for _, m := range regexp.MustCompile("`make ([\\w-]+)").FindAllStringSubmatch(content, -1) {
		targets = append(targets, m[1])
	}
	inFence := false
	command := regexp.MustCompile(`^\s*(?:\$ )?make ([\w-]+)`)
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if m := command.FindStringSubmatch(line); inFence && m != nil {
			targets = append(targets, m[1])
		}
	}
	return targets
}

func TestDocsMakeTargetsExist(t *testing.T) {
	t.Parallel()
	phony := regexp.MustCompile(`(?m)^\.PHONY:(.*)$`).FindStringSubmatch(readRepoFile(t, "Makefile"))
	if phony == nil {
		t.Fatal("Makefile has no .PHONY line")
	}
	targets := strings.Fields(phony[1])
	for _, doc := range []string{"README.md", "CONTRIBUTING.md", ".github/PULL_REQUEST_TEMPLATE.md"} {
		mentioned := docMakeTargets(readRepoFile(t, doc))
		if doc != ".github/PULL_REQUEST_TEMPLATE.md" && len(mentioned) == 0 {
			t.Errorf("%s mentions no make target", doc)
		}
		for _, target := range mentioned {
			if !slices.Contains(targets, target) {
				t.Errorf("%s mentions `make %s`, but the Makefile .PHONY targets are %v", doc, target, targets)
			}
		}
	}
}

// ciGates are the gate commands CI runs; CONTRIBUTING.md and the Makefile
// must document the same ones (constitution, Development Workflow).
var ciGates = []string{"gofmt -l", "go vet ./...", "go test -race"}

func TestContributingMatchesCIGates(t *testing.T) {
	t.Parallel()
	for _, doc := range []string{".github/workflows/ci.yml", "CONTRIBUTING.md", "Makefile"} {
		content := readRepoFile(t, doc)
		for _, gate := range ciGates {
			if !strings.Contains(content, gate) {
				t.Errorf("%s does not contain the gate %q", doc, gate)
			}
		}
	}
}

// inventedClaims are phrases that would name a channel, contact, or promise
// that does not exist (constitution Principle VI).
var inventedClaims = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bbrew (install|tap)\b`),
	regexp.MustCompile(`(?i)\bscoop (install|bucket)\b`),
	regexp.MustCompile(`(?i)\bwinget install\b`),
	regexp.MustCompile(`(?i)\bapt(-get)? install\b`),
	regexp.MustCompile(`(?i)\b(yay|paru) -S\b`),
	regexp.MustCompile(`(?i)\bchoco install\b`),
	regexp.MustCompile(`(?i)\bsnap install\b`),
	regexp.MustCompile(`(?i)\bguarantee`),
	regexp.MustCompile(`(?i)\bwithin \d+ (hours|days|business days)\b`),
	regexp.MustCompile(`\bSLA\b`),
	regexp.MustCompile(`(?i)\bbug bounty program\b`),
}

// placeholderEmailDomains are the only email domains (or parent domains)
// README examples may use: the reserved example.com and GitHub's noreply
// pattern.
var placeholderEmailDomains = []string{"example.com", "users.noreply.github.com"}

func TestDocsMakeNoInventedClaims(t *testing.T) {
	t.Parallel()
	docs := append(slices.Clone(linkedDocs), urlOnlyDocs...)
	for _, doc := range docs {
		content := readRepoFile(t, doc)
		for _, re := range inventedClaims {
			if m := re.FindString(content); m != "" {
				t.Errorf("%s contains %q; no such channel, promise, or program exists", doc, m)
			}
		}
		for _, email := range emailsIn(content) {
			_, domain, _ := strings.Cut(strings.ToLower(email), "@")
			if !slices.ContainsFunc(placeholderEmailDomains, func(d string) bool { return domain == d || strings.HasSuffix(domain, "."+d) }) {
				t.Errorf("%s names email %q; only placeholder domains %v are allowed", doc, email, placeholderEmailDomains)
			}
		}
	}
}

// TestSecuritySupportedVersionMatchesChangelog keeps the supported-version
// row of SECURITY.md on the newest released minor version.
func TestSecuritySupportedVersionMatchesChangelog(t *testing.T) {
	t.Parallel()
	newest := regexp.MustCompile(`(?m)^## \[(\d+)\.(\d+)\.\d+\]`).FindStringSubmatch(readRepoFile(t, "CHANGELOG.md"))
	if newest == nil {
		t.Fatal("CHANGELOG.md has no release heading")
	}
	want := "latest release (currently `v" + newest[1] + "." + newest[2] + ".x`)"
	if !strings.Contains(readRepoFile(t, "SECURITY.md"), want) {
		t.Fatalf("SECURITY.md does not contain %q; update the Supported versions table", want)
	}
}
