package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// An identity file is a ghs-owned Git config fragment holding one profile's
// user.name and user.email; the workspace includeIf entry points at it.

// IdentityFilePath is <config dir>/gitconfig-<profile>.
func IdentityFilePath(configPath string, profileName string) string {
	return filepath.Join(filepath.Dir(configPath), "gitconfig-"+profileName)
}

func identityContent(name, email string) string {
	return "[user]\n\tname = " + gitConfigValue(name) + "\n\temail = " + gitConfigValue(email) + "\n"
}

// gitConfigValue quotes a value when Git would otherwise misread it
// (comment characters, backslashes, surrounding whitespace). Values never
// contain double quotes or control characters (validated earlier).
func gitConfigValue(v string) string {
	if strings.ContainsAny(v, `#;\`) || strings.TrimSpace(v) != v {
		return `"` + strings.ReplaceAll(v, `\`, `\\`) + `"`
	}
	return v
}

// WriteIdentityFile writes the identity file atomically with mode 0600. An
// existing file with the same content is left untouched (changed == false).
func WriteIdentityFile(path string, name string, email string) (changed bool, err error) {
	content := identityContent(name, email)
	if existing, err := os.ReadFile(path); err == nil && string(existing) == content {
		return false, nil
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return false, fmt.Errorf("read identity file %s: %w", path, err)
	}
	if err := writeFileAtomic(path, []byte(content), 0o600, nil); err != nil {
		return false, fmt.Errorf("write identity file %s: %w", path, err)
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return false, fmt.Errorf("write identity file %s: %w", path, err)
	}
	return true, nil
}

// ReadIdentityFile returns the name and email recorded in an identity file.
func ReadIdentityFile(path string) (name string, email string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	inUser := false
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if strings.HasPrefix(line, "[") {
			inUser = strings.EqualFold(strings.Trim(line, "[] "), "user")
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || !inUser {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			value = strings.ReplaceAll(value[1:len(value)-1], `\\`, `\`)
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "name":
			name = value
		case "email":
			email = value
		}
	}
	return name, email, nil
}

// IdentityFileUpToDate reports whether the file exists with exactly this
// name and email.
func IdentityFileUpToDate(path string, name string, email string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return string(data) == identityContent(name, email), nil
}

// RemoveIdentityFile deletes the identity file; an absent file is success
// with removed == false.
func RemoveIdentityFile(path string) (removed bool, err error) {
	if err := os.Remove(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("delete identity file %s: %w", path, err)
	}
	return true, nil
}
