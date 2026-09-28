package main

import (
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
