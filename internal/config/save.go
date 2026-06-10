package config

import (
	"bufio"
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

	bw := bufio.NewWriter(file)
	// writePair writes key = "value"\n without passing values through fmt
	// interface (which would cause heap escapes). Errors accumulate in bw
	// and are returned on Flush.
	writePair := func(key, value string) {
		if value == "" {
			return
		}
		bw.WriteString(key)
		bw.WriteString(` = "`)
		bw.WriteString(value)
		bw.WriteString("\"\n")
	}

	for i, profile := range cfg.Profiles {
		if i > 0 {
			bw.WriteByte('\n')
		}
		bw.WriteByte('[')
		bw.WriteString(profile.Name)
		bw.WriteString("]\n")
		writePair("gh_user", profile.GitHubUser)
		writePair("git_name", profile.GitName)
		writePair("git_email", profile.GitEmail)
		writePair("ssh_host_alias", profile.SSHHostAlias)
		writePair("ssh_key", profile.SSHKey)
		writePair("workspace", profile.Workspace)
	}
	if err := bw.Flush(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}
