package config

import (
	"fmt"
	"os"
	"path/filepath"
)

func Save(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open config for write: %w", err)
	}
	defer func() { _ = file.Close() }()

	writePair := func(key, value string) error {
		if value == "" {
			return nil
		}
		_, err := fmt.Fprintf(file, "%s = %q\n", key, value)
		return err
	}
	for i, profile := range cfg.Profiles {
		if i > 0 {
			if _, err := fmt.Fprintln(file); err != nil {
				return fmt.Errorf("write config: %w", err)
			}
		}
		if _, err := fmt.Fprintf(file, "[%s]\n", profile.Name); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := writePair("gh_user", profile.GitHubUser); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := writePair("git_name", profile.GitName); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := writePair("git_email", profile.GitEmail); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := writePair("ssh_host_alias", profile.SSHHostAlias); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := writePair("ssh_key", profile.SSHKey); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
		if err := writePair("workspace", profile.Workspace); err != nil {
			return fmt.Errorf("write config: %w", err)
		}
	}
	return nil
}
