package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.conf")
	content := `[work]
gh_user = "zamyb"
git_name = "IZZAMUDDIN"
git_email = "work@example.com"
ssh_host_alias = "github-work"
ssh_key = "~/.ssh/id_ed25519_work"
workspace = "~/Documents/work"
`
	if err := osWriteFile(path, content); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	profile, ok := cfg.FindProfile("work")
	if !ok {
		t.Fatal("FindProfile() did not find work")
	}
	if profile.GitHubUser != "zamyb" || profile.GitEmail != "work@example.com" || profile.SSHHostAlias != "github-work" {
		t.Fatalf("profile = %+v", profile)
	}
}

func osWriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
