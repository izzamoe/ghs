package sshops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/runner"
)

type SSH struct {
	runner runner.Runner
}

func New(run runner.Runner) SSH {
	return SSH{runner: run}
}

func (s SSH) EnsureKey(profile config.Profile) error {
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return err
	}
	if _, err := os.Stat(keyPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check ssh key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return fmt.Errorf("create ssh directory: %w", err)
	}
	comment := profile.GitHubUser
	if comment == "" {
		comment = profile.Name
	}
	return s.runner.Run("ssh-keygen", "-t", "ed25519", "-C", comment, "-f", keyPath, "-N", "")
}

func (s SSH) EnsureConfig(profile config.Profile) error {
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	sshConfigPath := filepath.Join(home, ".ssh", "config")
	if err := os.MkdirAll(filepath.Dir(sshConfigPath), 0o700); err != nil {
		return fmt.Errorf("create ssh directory: %w", err)
	}

	contentBytes, err := os.ReadFile(sshConfigPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read ssh config: %w", err)
	}
	content := string(contentBytes)
	if hasHostBlock(content, profile.SSHHostAlias) {
		return nil
	}

	block := fmt.Sprintf("\nHost %s\n  HostName github.com\n  User git\n  IdentityFile %s\n  IdentitiesOnly yes\n", profile.SSHHostAlias, filepath.ToSlash(keyPath))
	file, err := os.OpenFile(sshConfigPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open ssh config: %w", err)
	}
	defer func() { _ = file.Close() }()
	if _, err := file.WriteString(block); err != nil {
		return fmt.Errorf("write ssh config: %w", err)
	}
	return nil
}

func (s SSH) UploadKey(profile config.Profile) error {
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return err
	}
	title := "ghs-" + profile.Name
	return s.runner.Run("gh", "ssh-key", "add", keyPath+".pub", "--title", title)
}

func hasHostBlock(content string, alias string) bool {
	for line := range strings.SplitSeq(content, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 6 || !strings.EqualFold(line[:5], "Host ") {
			continue
		}
		for word := range strings.FieldsSeq(line[5:]) {
			if word == alias {
				return true
			}
		}
	}
	return false
}
