package gitops

import (
	"strings"
)

// A workspace link is a global Git entry
//
//	includeIf.gitdir/i:<pattern>.path = <identity file>
//
// added and removed only through `git config --global` (FR-054, FR-055).
// gitdir/i matches case-insensitively, which is what macOS and Windows file
// systems need; a trailing "/" makes Git match everything below it.

// WorkspacePattern turns a workspace directory into an includeIf pattern:
// forward slashes, "~/" kept for Git to expand, exactly one trailing "/".
func WorkspacePattern(workspace string) string {
	return strings.TrimRight(strings.ReplaceAll(workspace, `\`, "/"), "/") + "/"
}

// IncludeIfKey is the Git config key for a workspace pattern.
func IncludeIfKey(pattern string) string {
	return "includeIf.gitdir/i:" + pattern + ".path"
}

// HasInclude reports whether the global key lists file. A missing key (exit
// 1) is "not present"; any other failure is an error.
func (g Git) HasInclude(key string, file string) (bool, error) {
	out, err := g.runner.Output("git", "config", "--global", "--get-all", key)
	if err != nil {
		if exitCode(err) == 1 {
			return false, nil
		}
		return false, err
	}
	for line := range strings.SplitSeq(out, "\n") {
		if strings.TrimSpace(line) == file {
			return true, nil
		}
	}
	return false, nil
}

// AddInclude appends file to the global key, leaving other values alone.
func (g Git) AddInclude(key string, file string) error {
	return g.runner.Run("git", "config", "--global", "--add", key, file)
}

// RemoveInclude removes exactly the value file from the global key
// (--fixed-value, so the path is not a regular expression). An absent value
// (exit 5) is success with removed == false.
func (g Git) RemoveInclude(key string, file string) (removed bool, err error) {
	err = g.runner.Run("git", "config", "--global", "--unset", "--fixed-value", key, file)
	if err != nil {
		if exitCode(err) == 5 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
