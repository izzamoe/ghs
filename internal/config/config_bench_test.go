package config

import (
	"os"
	"path/filepath"
	"testing"
)

const benchConfig = `[work]
gh_user = "zamyb"
git_name = "IZZAMUDDIN"
git_email = "work@example.com"
ssh_host_alias = "github-zamyb"
ssh_key = "~/.ssh/id_ed25519_zamyb"
workspace = "~/Documents/work"

[me]
gh_user = "izzamoe"
git_name = "IZZAMUDDIN"
git_email = "dimanton1221@gmail.com"
ssh_host_alias = "github-izzamoe"
ssh_key = "~/.ssh/id_ed25519_izzamoe"
`

func BenchmarkLoad(b *testing.B) {
	path := filepath.Join(b.TempDir(), "config.conf")
	if err := os.WriteFile(path, []byte(benchConfig), 0o600); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		if _, err := Load(path); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSave(b *testing.B) {
	cfg := Config{Profiles: []Profile{
		{Name: "work", GitHubUser: "zamyb", GitName: "IZZAMUDDIN", GitEmail: "work@example.com", SSHHostAlias: "github-zamyb", SSHKey: "~/.ssh/id_ed25519_zamyb", Workspace: "~/Documents/work"},
		{Name: "me", GitHubUser: "izzamoe", GitName: "IZZAMUDDIN", GitEmail: "dimanton1221@gmail.com", SSHHostAlias: "github-izzamoe", SSHKey: "~/.ssh/id_ed25519_izzamoe"},
	}}
	path := filepath.Join(b.TempDir(), "config.conf")
	b.ResetTimer()
	for b.Loop() {
		if err := Save(path, cfg); err != nil {
			b.Fatal(err)
		}
	}
}
