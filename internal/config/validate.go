package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The validators below implement FR-007 to FR-012, FR-056, and FR-061 (see
// specs/001-ghs-context-safety/data-model.md §1). They apply identically to
// values from flags, from the GitHub CLI, and from the config file. Each
// error names the field and the rule that was broken.

// ValidateName checks a profile name. reserved lists the command names a
// profile may not shadow; it is passed in to avoid an import cycle with cli.
func ValidateName(name string, reserved []string) error {
	fail := func(rule string) error { return fmt.Errorf("invalid profile name %q: %s", name, rule) }
	if name == "" {
		return fmt.Errorf("invalid profile name: must not be empty")
	}
	if n := utf8.RuneCountInString(name); n > 64 {
		return fail("must be at most 64 characters")
	}
	if name[0] == '-' || name[0] == '.' {
		return fail("must not start with - or .")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '.' && r != '_' && r != '-' {
			return fail("only letters, digits, '.', '_', and '-' are allowed")
		}
	}
	for _, word := range reserved {
		if strings.EqualFold(name, word) {
			return fail("it is a ghs command name")
		}
	}
	return nil
}

var aliasPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// ValidateAlias checks an SSH host alias.
func ValidateAlias(alias string) error {
	fail := func(rule string) error { return fmt.Errorf("invalid ssh alias %q: %s", alias, rule) }
	switch {
	case alias == "":
		return fmt.Errorf("invalid ssh alias: must not be empty")
	case !aliasPattern.MatchString(alias):
		return fail("only letters, digits, '.', '_', and '-' are allowed (1 to 64 characters)")
	case alias[0] == '-':
		return fail("must not start with -")
	case strings.EqualFold(alias, "github.com"):
		return fail("must not be github.com, which would shadow the real host")
	}
	return nil
}

var loginPattern = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)

// ValidateLogin checks a GitHub login against GitHub's own rules.
func ValidateLogin(login string) error {
	if login == "" {
		return fmt.Errorf("invalid github login: must not be empty")
	}
	if len(login) > 39 || !loginPattern.MatchString(login) {
		return fmt.Errorf("invalid github login %q: use letters, digits, and single hyphens, not at the start or end, at most 39 characters", login)
	}
	return nil
}

// ValidateGitName checks a Git author name.
func ValidateGitName(name string) error {
	switch {
	case strings.TrimSpace(name) == "":
		return fmt.Errorf("invalid git name: must not be empty")
	case hasControl(name):
		return fmt.Errorf("invalid git name %q: must not contain control characters", name)
	case strings.Contains(name, `"`):
		return fmt.Errorf("invalid git name %q: must not contain a double quote", name)
	}
	return nil
}

// ValidateEmail checks a Git author email.
func ValidateEmail(email string) error {
	fail := func(rule string) error { return fmt.Errorf("invalid email %q: %s", email, rule) }
	if email == "" {
		return fmt.Errorf("invalid email: must not be empty")
	}
	for _, r := range email {
		if unicode.IsSpace(r) || r < 0x20 || r == 0x7f || r == '"' {
			return fail("must not contain whitespace, control characters, or double quotes")
		}
	}
	if strings.Count(email, "@") != 1 {
		return fail("must contain exactly one @")
	}
	local, domain, _ := strings.Cut(email, "@")
	if local == "" {
		return fail("the part before @ must not be empty")
	}
	if !strings.Contains(domain, ".") {
		return fail("the domain must contain a dot")
	}
	for label := range strings.SplitSeq(domain, ".") {
		if label == "" {
			return fail("the domain must not have empty labels")
		}
	}
	return nil
}

// ValidateKeyPath checks an SSH private key path.
func ValidateKeyPath(path string) error {
	switch {
	case path == "":
		return fmt.Errorf("invalid ssh key path: must not be empty")
	case hasControl(path) || strings.Contains(path, `"`):
		return fmt.Errorf("invalid ssh key path %q: must not contain control characters or double quotes", path)
	case !isAbsLike(path) || path == "~":
		return fmt.Errorf("invalid ssh key path %q: must be absolute or start with ~/", path)
	}
	return nil
}

var driveRoot = regexp.MustCompile(`^[A-Za-z]:$`)

// ValidateWorkspace checks a workspace directory.
func ValidateWorkspace(path string) error {
	switch {
	case path == "":
		return fmt.Errorf("invalid workspace: must not be empty")
	case hasControl(path) || strings.Contains(path, `"`):
		return fmt.Errorf("invalid workspace %q: must not contain control characters or double quotes", path)
	case !isAbsLike(path):
		return fmt.Errorf("invalid workspace %q: must be absolute or start with ~/", path)
	}
	norm := NormalizeWorkspace(path)
	if norm == "" || norm == "~" || driveRoot.MatchString(norm) || isHomeDir(norm) {
		return fmt.Errorf("invalid workspace %q: must not be the home directory or a filesystem root", path)
	}
	return nil
}

// NormalizeWorkspace converts backslashes to forward slashes and removes
// trailing slashes; workspaces are stored and matched in this form.
func NormalizeWorkspace(path string) string {
	return strings.TrimRight(strings.ReplaceAll(path, `\`, "/"), "/")
}

// isAbsLike reports whether path is absolute on any supported platform or
// starts with ~/. Drive-letter paths are accepted everywhere so that the
// same config validates identically on every platform.
func isAbsLike(path string) bool {
	p := strings.ReplaceAll(path, `\`, "/")
	switch {
	case strings.HasPrefix(p, "~/"), strings.HasPrefix(p, "/"):
		return true
	case len(p) >= 3 && p[1] == ':' && p[2] == '/' && isASCIILetter(p[0]):
		return true
	}
	return filepath.IsAbs(path)
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isHomeDir(norm string) bool {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return false
	}
	return strings.EqualFold(norm, NormalizeWorkspace(home))
}

// comparableWorkspace expands ~/ and lower-cases a workspace so that equal
// and nested directories can be detected (Git matches gitdir/i
// case-insensitively).
func comparableWorkspace(path string) string {
	norm := NormalizeWorkspace(path)
	if rest, ok := strings.CutPrefix(norm, "~/"); ok {
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			norm = NormalizeWorkspace(home) + "/" + rest
		}
	}
	return strings.ToLower(norm) + "/"
}

// CheckUnique reports a collision between p and any other profile (the one
// at ignoreIndex is the profile being replaced and is skipped): names and
// aliases are unique ignoring case; workspaces must not be equal or nested.
func (c Config) CheckUnique(p Profile, ignoreIndex int) error {
	for i, other := range c.Profiles {
		if i == ignoreIndex {
			continue
		}
		if strings.EqualFold(other.Name, p.Name) {
			if other.Name == p.Name {
				return fmt.Errorf("profile %q already exists", other.Name)
			}
			return fmt.Errorf("profile name %q conflicts with existing profile %q (names are case-insensitive); use %q", p.Name, other.Name, other.Name)
		}
		if p.SSHHostAlias != "" && strings.EqualFold(other.SSHHostAlias, p.SSHHostAlias) {
			return fmt.Errorf("ssh alias %q is already used by profile %q; pass --ssh-alias with a different alias", p.SSHHostAlias, other.Name)
		}
		if p.Workspace != "" && other.Workspace != "" {
			a, b := comparableWorkspace(p.Workspace), comparableWorkspace(other.Workspace)
			if strings.HasPrefix(a, b) || strings.HasPrefix(b, a) {
				return fmt.Errorf("workspace %s overlaps profile %q (%s)", NormalizeWorkspace(p.Workspace), other.Name, other.Workspace)
			}
		}
	}
	return nil
}

var unsafeSuffixChars = regexp.MustCompile(`[^a-z0-9._-]+`)

// SafeSuffix derives the lower-case ASCII suffix used for default aliases
// (github-<suffix>) and key paths (~/.ssh/id_ed25519_<suffix>).
func SafeSuffix(name string) string {
	cleaned := strings.Trim(unsafeSuffixChars.ReplaceAllString(strings.ToLower(name), "-"), "-.")
	if cleaned == "" {
		return "profile"
	}
	return cleaned
}
