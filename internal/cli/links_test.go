package cli

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"unicode"
)

// allowedLinkHosts are the only hosts an absolute link in the documentation
// set may point at (constitution Principle VI). Adding a host is a reviewed
// change; the test never fetches anything.
var allowedLinkHosts = []string{
	"github.com",
	"pkg.go.dev",
	"go.dev",
	"cli.github.com",
	"docs.github.com",
	"keepachangelog.com",
	"semver.org",
	"www.contributor-covenant.org",
}

// linkedDocs are the Markdown documents whose links are checked.
var linkedDocs = []string{"README.md", "CONTRIBUTING.md", "SECURITY.md", "SUPPORT.md", "CODE_OF_CONDUCT.md", "CHANGELOG.md"}

// urlOnlyDocs are scanned for absolute URLs only.
var urlOnlyDocs = []string{
	".github/PULL_REQUEST_TEMPLATE.md",
	".github/ISSUE_TEMPLATE/bug_report.yml",
	".github/ISSUE_TEMPLATE/feature_request.yml",
	".github/ISSUE_TEMPLATE/config.yml",
}

var (
	inlineLink    = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	referenceLink = regexp.MustCompile(`(?m)^\[[^\]]+\]:\s*(\S+)`)
	absoluteURL   = regexp.MustCompile("https?://[^\\s)<>\\]\"'`]+")
)

// stripFences returns content without fenced code blocks, so examples such
// as "# comment" lines are not taken for headings or links.
func stripFences(content string) string {
	var b strings.Builder
	inFence := false
	for line := range strings.SplitSeq(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// docLinks returns every inline and reference-style link target outside
// fenced code blocks.
func docLinks(t *testing.T, name string) []string {
	t.Helper()
	prose := stripFences(readRepoFile(t, name))
	var links []string
	for _, m := range inlineLink.FindAllStringSubmatch(prose, -1) {
		links = append(links, m[1])
	}
	for _, m := range referenceLink.FindAllStringSubmatch(prose, -1) {
		links = append(links, m[1])
	}
	return links
}

// headingAnchor is the anchor GitHub generates for a heading: lower case,
// punctuation dropped (including backticks), spaces turned into hyphens.
func headingAnchor(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteByte('-')
		}
	}
	return b.String()
}

// docAnchors returns the anchors of every Markdown heading in name,
// numbering duplicates the way GitHub does.
func docAnchors(t *testing.T, name string) []string {
	t.Helper()
	seen := map[string]int{}
	var anchors []string
	for line := range strings.SplitSeq(stripFences(readRepoFile(t, name)), "\n") {
		trimmed := strings.TrimLeft(line, "#")
		if trimmed == line || !strings.HasPrefix(trimmed, " ") {
			continue
		}
		anchor := headingAnchor(strings.TrimSpace(trimmed))
		if n := seen[anchor]; n > 0 {
			anchors = append(anchors, anchor+"-"+strconv.Itoa(n))
		} else {
			anchors = append(anchors, anchor)
		}
		seen[anchor]++
	}
	return anchors
}

func TestHeadingAnchor(t *testing.T) {
	t.Parallel()
	for heading, want := range map[string]string{
		"What ghs does":          "what-ghs-does",
		"Option A: Go toolchain": "option-a-go-toolchain",
		"Switching: `use`":       "switching-use",
		"Seeing where you are: `list`, `status`, `doctor`": "seeing-where-you-are-list-status-doctor",
		"`ghs add-profile`": "ghs-add-profile",
	} {
		if got := headingAnchor(heading); got != want {
			t.Errorf("headingAnchor(%q) = %q, want %q", heading, got, want)
		}
	}
}

func TestDocLinksResolve(t *testing.T) {
	t.Parallel()
	for _, doc := range linkedDocs {
		for _, link := range docLinks(t, doc) {
			if strings.HasPrefix(link, "http://") || strings.HasPrefix(link, "https://") {
				continue // host checked by TestDocURLHostsAllowed
			}
			target, fragment, _ := strings.Cut(link, "#")
			if target == "" {
				target = doc
			}
			if _, err := os.Stat(filepath.Join("..", "..", filepath.FromSlash(target))); err != nil {
				t.Errorf("%s links to %q, which does not exist", doc, link)
				continue
			}
			if fragment != "" && strings.HasSuffix(target, ".md") && !slices.Contains(docAnchors(t, target), fragment) {
				t.Errorf("%s links to %q, but %s has no heading with anchor #%s", doc, link, target, fragment)
			}
		}
	}
}

func TestDocURLHostsAllowed(t *testing.T) {
	t.Parallel()
	for _, doc := range append(slices.Clone(linkedDocs), urlOnlyDocs...) {
		for _, raw := range absoluteURL.FindAllString(readRepoFile(t, doc), -1) {
			u, err := url.Parse(strings.TrimRight(raw, ".,;:"))
			if err != nil {
				t.Errorf("%s: cannot parse URL %q: %v", doc, raw, err)
				continue
			}
			if u.Scheme != "https" {
				t.Errorf("%s: URL %q is not https", doc, raw)
			}
			if !slices.Contains(allowedLinkHosts, u.Host) {
				t.Errorf("%s: URL %q uses host %q, which is not in allowedLinkHosts (internal/cli/links_test.go)", doc, raw, u.Host)
			}
		}
	}
}
