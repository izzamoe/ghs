package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
)

// Load reads and strictly parses the config file. A missing file returns an
// empty Config and an error satisfying errors.Is(err, fs.ErrNotExist); every
// other problem (unreadable file, malformed line) is an error that callers
// must not treat as an empty config.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Config{}, fmt.Errorf("config %s does not exist: %w", path, err)
		}
		return Config{}, fmt.Errorf("cannot read config %s: %w", path, err)
	}
	cfg, err := parse(string(data))
	if err != nil {
		return Config{}, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return cfg, nil
}

type parseError struct {
	line int
	msg  string
}

func (e *parseError) Error() string {
	return fmt.Sprintf("line %d: %s", e.line, e.msg)
}

func lineErr(line int, format string, a ...any) error {
	return &parseError{line: line, msg: fmt.Sprintf(format, a...)}
}

func parse(content string) (Config, error) {
	var cfg Config
	var current *Profile
	seen := map[string]int{} // lower-cased name -> line number
	lineNo := 0
	for raw := range strings.SplitSeq(content, "\n") {
		lineNo++
		line := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		if line[0] == '[' {
			if line[len(line)-1] != ']' {
				return Config{}, lineErr(lineNo, "section header is missing a closing ]")
			}
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				return Config{}, lineErr(lineNo, "empty profile name")
			}
			if hasControl(name) {
				return Config{}, lineErr(lineNo, "control character in profile name")
			}
			key := strings.ToLower(name)
			if first, ok := seen[key]; ok {
				return Config{}, lineErr(lineNo, "duplicate profile %q (first defined on line %d; names are case-insensitive)", name, first)
			}
			seen[key] = lineNo
			cfg.Profiles = append(cfg.Profiles, Profile{Name: name})
			current = &cfg.Profiles[len(cfg.Profiles)-1]
			continue
		}
		if current == nil {
			return Config{}, lineErr(lineNo, "key outside a profile section")
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, lineErr(lineNo, "expected key = value")
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.ContainsAny(key, " \t\"[]") {
			return Config{}, lineErr(lineNo, "invalid key %q", key)
		}
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, `"`) {
			if len(value) < 2 || !strings.HasSuffix(value, `"`) {
				return Config{}, lineErr(lineNo, "unterminated quote")
			}
			value = value[1 : len(value)-1]
		}
		if hasControl(value) {
			return Config{}, lineErr(lineNo, "control character in value of %s", key)
		}
		setProfileValue(current, key, value)
	}
	return cfg, nil
}

func setProfileValue(profile *Profile, key string, value string) {
	switch key {
	case "gh_user":
		profile.GitHubUser = value
	case "git_name":
		profile.GitName = value
	case "git_email":
		profile.GitEmail = value
	case "ssh_host_alias":
		profile.SSHHostAlias = value
	case "ssh_key":
		profile.SSHKey = value
	case "workspace":
		profile.Workspace = value
	default:
		profile.Extra = append(profile.Extra, KeyValue{Key: key, Value: value})
	}
}

func hasControl(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return true
		}
	}
	return false
}
