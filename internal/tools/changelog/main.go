// Command changelog prints the CHANGELOG.md section for a release tag, for
// use as GitHub Release notes:
//
//	go run ./internal/tools/changelog -tag v0.5.0 -file CHANGELOG.md
//
// It is a build-time helper (see specs/001-ghs-context-safety/plan.md,
// Complexity Tracking) and is not part of the ghs binary.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	tag := flag.String("tag", "", "release tag such as v0.5.0, or Unreleased")
	file := flag.String("file", "CHANGELOG.md", "changelog path")
	flag.Parse()
	if *tag == "" {
		fmt.Fprintln(os.Stderr, "changelog: -tag is required")
		os.Exit(2)
	}
	data, err := os.ReadFile(*file)
	if err != nil {
		fmt.Fprintln(os.Stderr, "changelog:", err)
		os.Exit(1)
	}
	section, err := extractSection(strings.ReplaceAll(string(data), "\r\n", "\n"), *tag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "changelog:", err)
		os.Exit(1)
	}
	fmt.Println(section)
}

// extractSection returns the body of the "## [<version>]" section (Keep a
// Changelog format) for tag, without its heading, the next section, or the
// link reference definitions at the end of the file.
func extractSection(content string, tag string) (string, error) {
	version := strings.TrimPrefix(tag, "v")
	heading := "## [" + version + "]"
	var body []string
	found := false
	for line := range strings.SplitSeq(content, "\n") {
		if !found {
			found = line == heading || strings.HasPrefix(line, heading+" ")
			continue
		}
		if strings.HasPrefix(line, "## [") || (strings.HasPrefix(line, "[") && strings.Contains(line, "]: ")) {
			break
		}
		body = append(body, line)
	}
	if !found {
		return "", fmt.Errorf("no %q section in the changelog", heading)
	}
	section := strings.TrimSpace(strings.Join(body, "\n"))
	if section == "" {
		return "", errors.New("section " + heading + " is empty")
	}
	return section, nil
}
