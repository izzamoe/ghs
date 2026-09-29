package main

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const sample = `# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- ` + "`doctor`" + `

## [0.3.0] - 2026-01-02

### Fixed

- a bug

## [0.2.0] - 2025-12-01

- older

[Unreleased]: https://github.com/izzamoe/ghs/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/izzamoe/ghs/releases/tag/v0.3.0
`

func TestExtractSectionForTag(t *testing.T) {
	t.Parallel()

	got, err := extractSection(sample, "v0.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "### Fixed\n\n- a bug" {
		t.Fatalf("v0.3.0 section = %q", got)
	}

	got, err = extractSection(sample, "Unreleased")
	if err != nil || got != "### Added\n\n- `doctor`" {
		t.Fatalf("Unreleased section = %q, %v", got, err)
	}

	got, err = extractSection(sample, "0.2.0")
	if err != nil || got != "- older" {
		t.Fatalf("0.2.0 section = %q, %v (link references must not leak in)", got, err)
	}

	if _, err := extractSection(sample, "v9.9.9"); err == nil || !strings.Contains(err.Error(), "9.9.9") {
		t.Fatalf("missing section error = %v", err)
	}
	if _, err := extractSection("## [1.0.0]\n\n", "v1.0.0"); err == nil {
		t.Fatal("empty section accepted")
	}
}

// TestChangelogLinkReferences keeps the repository CHANGELOG.md link
// references in step with its release headings (Keep a Changelog): every
// "## [x.y.z]" has exactly one release-tag reference, no reference lacks a
// heading, and [Unreleased] compares from the newest section's tag.
func TestChangelogLinkReferences(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("../../../CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	content := strings.ReplaceAll(string(data), "\r\n", "\n")

	var headings []string
	for _, m := range regexp.MustCompile(`(?m)^## \[(\d+\.\d+\.\d+)\]`).FindAllStringSubmatch(content, -1) {
		headings = append(headings, m[1])
	}
	if len(headings) == 0 {
		t.Fatal("CHANGELOG.md has no release headings")
	}
	var refs []string
	for _, m := range regexp.MustCompile(`(?m)^\[(\d+\.\d+\.\d+)\]: (.*)$`).FindAllStringSubmatch(content, -1) {
		want := "https://github.com/izzamoe/ghs/releases/tag/v" + m[1]
		if m[2] != want {
			t.Errorf("[%s] reference is %q, want %q", m[1], m[2], want)
		}
		if slices.Contains(refs, m[1]) {
			t.Errorf("[%s] reference appears more than once", m[1])
		}
		refs = append(refs, m[1])
	}
	for _, v := range headings {
		if !slices.Contains(refs, v) {
			t.Errorf("section ## [%s] has no link reference; add: [%s]: https://github.com/izzamoe/ghs/releases/tag/v%s", v, v, v)
		}
	}
	for _, v := range refs {
		if !slices.Contains(headings, v) {
			t.Errorf("link reference [%s] has no ## [%s] section", v, v)
		}
	}
	unreleased := "[Unreleased]: https://github.com/izzamoe/ghs/compare/v" + headings[0] + "...HEAD"
	if !slices.Contains(strings.Split(content, "\n"), unreleased) {
		t.Errorf("CHANGELOG.md lacks the line %q (the newest section is %s)", unreleased, headings[0])
	}
}
