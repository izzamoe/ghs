package config

import (
	"fmt"
	"os"
	"strings"
)

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}

	// string(data) is one alloc; all substrings derived from it share the
	// same backing memory — no per-line or per-field copies needed.
	content := string(data)
	var cfg Config
	var current *Profile
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' && line[len(line)-1] == ']' {
			name := strings.TrimSpace(line[1 : len(line)-1])
			if name == "" {
				return Config{}, fmt.Errorf("empty profile name")
			}
			cfg.Profiles = append(cfg.Profiles, Profile{Name: name})
			current = &cfg.Profiles[len(cfg.Profiles)-1]
			continue
		}
		if current == nil {
			return Config{}, fmt.Errorf("config key outside profile: %s", line)
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			return Config{}, fmt.Errorf("invalid config line: %s", line)
		}
		setProfileValue(current, strings.TrimSpace(key), strings.Trim(value, " \t\r\n\""))
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
	}
}
