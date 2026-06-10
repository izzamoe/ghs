package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Load(path string) (Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer func() { _ = file.Close() }()

	var cfg Config
	var current *Profile
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
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
		setProfileValue(current, strings.TrimSpace(key), strings.Trim(strings.TrimSpace(value), `"`))
	}
	if err := scanner.Err(); err != nil {
		return Config{}, fmt.Errorf("scan config: %w", err)
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
