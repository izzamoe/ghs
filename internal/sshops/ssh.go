package sshops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
)

// commandRunner is the subset of runner.Runner that SSH needs.
type commandRunner interface {
	Run(name string, args ...string) error
}

type SSH struct {
	runner commandRunner
}

func New(run commandRunner) SSH {
	return SSH{runner: run}
}

// ConfigPath returns the user's SSH client config path (~/.ssh/config).
func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

// ReadConfigFile returns the SSH config content; a missing file is empty,
// any other read error is returned.
func ReadConfigFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("cannot read ssh config %s: %w", path, err)
	}
	return string(data), nil
}

// CheckKeyPair reports whether the private key exists. A private key without
// its .pub sibling is an error: ghs never regenerates or overwrites it.
func CheckKeyPair(keyPath string) (exists bool, err error) {
	if _, err := os.Stat(keyPath); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("check ssh key %s: %w", keyPath, err)
	}
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return true, fmt.Errorf("private key %s exists but %s.pub is missing; recreate it with: ssh-keygen -y -f %s > %s.pub", keyPath, keyPath, keyPath, keyPath)
		}
		return true, fmt.Errorf("check ssh public key %s.pub: %w", keyPath, err)
	}
	return true, nil
}

// EnsureKey generates the profile's key pair when it does not exist yet.
func (s SSH) EnsureKey(profile config.Profile) (keyPath string, created bool, err error) {
	keyPath, err = config.ExpandPath(profile.SSHKey)
	if err != nil {
		return "", false, err
	}
	comment := profile.GitHubUser
	if comment == "" {
		comment = profile.Name
	}
	created, err = s.EnsureKeyAt(keyPath, comment)
	return keyPath, created, err
}

// EnsureKeyAt runs ssh-keygen for keyPath unless the pair already exists.
// It never overwrites an existing private key.
func (s SSH) EnsureKeyAt(keyPath string, comment string) (created bool, err error) {
	exists, err := CheckKeyPair(keyPath)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return false, fmt.Errorf("create ssh directory: %w", err)
	}
	if err := s.runner.Run("ssh-keygen", "-t", "ed25519", "-C", comment, "-f", keyPath, "-N", ""); err != nil {
		return false, err
	}
	return true, nil
}

// EnsureConfig appends the profile's Host block to ~/.ssh/config when no
// Host line names the alias yet.
func EnsureConfig(profile config.Profile) (path string, added bool, err error) {
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return "", false, err
	}
	path, err = ConfigPath()
	if err != nil {
		return "", false, err
	}
	added, err = EnsureConfigAt(path, profile.SSHHostAlias, keyPath)
	return path, added, err
}

// EnsureConfigAt reads the SSH config first (an unreadable file is an
// error) and, only if no Host line lists alias, appends a block starting on
// its own line. The file is never rewritten.
func EnsureConfigAt(path string, alias string, keyPath string) (added bool, err error) {
	content, err := ReadConfigFile(path)
	if err != nil {
		return false, err
	}
	if hasHostBlock(content, alias) {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("create ssh directory: %w", err)
	}

	var prefix string
	switch {
	case content == "":
	case strings.HasSuffix(content, "\n"):
		prefix = "\n"
	default:
		prefix = "\n\n"
	}
	block := prefix + "Host " + alias + "\n  HostName github.com\n  User git\n  IdentityFile " +
		quoteIfNeeded(filepath.ToSlash(keyPath)) + "\n  IdentitiesOnly yes\n"

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return false, fmt.Errorf("open ssh config: %w", err)
	}
	if _, err := file.WriteString(block); err != nil {
		_ = file.Close()
		return false, fmt.Errorf("write ssh config: %w", err)
	}
	if err := file.Close(); err != nil {
		return false, fmt.Errorf("write ssh config: %w", err)
	}
	return true, nil
}

// quoteIfNeeded wraps a path containing whitespace in double quotes, which
// OpenSSH requires; paths never contain quotes (validated earlier).
func quoteIfNeeded(p string) string {
	if strings.ContainsAny(p, " \t") {
		return `"` + p + `"`
	}
	return p
}

// UploadKey adds the profile's public key (never the private key) to the
// active GitHub CLI account. A key that is already registered is reported
// through alreadyPresent.
func UploadKey(gh ghops.GH, profile config.Profile) (pubPath string, alreadyPresent bool, err error) {
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return "", false, err
	}
	pubPath = keyPath + ".pub"
	alreadyPresent, err = gh.AddSSHKey(pubPath, "ghs-"+profile.Name)
	return pubPath, alreadyPresent, err
}

// hasHostBlock reports whether any Host line lists alias (case-insensitive,
// alone or among several patterns, "Host alias" or "Host=alias").
func hasHostBlock(content string, alias string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		keyword, rest := splitKeyword(line)
		if !strings.EqualFold(keyword, "Host") {
			continue
		}
		for word := range strings.FieldsSeq(rest) {
			if strings.EqualFold(strings.Trim(word, `"`), alias) {
				return true
			}
		}
	}
	return false
}

// splitKeyword splits an ssh_config line into its keyword and the rest,
// accepting whitespace or "=" as the separator.
func splitKeyword(line string) (keyword string, rest string) {
	line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
	if line == "" || line[0] == '#' {
		return "", ""
	}
	end := strings.IndexAny(line, " \t=")
	if end < 0 {
		return line, ""
	}
	keyword = line[:end]
	rest = strings.TrimSpace(line[end:])
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
	return keyword, rest
}
