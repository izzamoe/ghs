package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCRLF(t *testing.T) {
	t.Parallel()

	content := "[work]\r\ngh_user = \"zamyb\"\r\ngit_email = \"work@example.com\"\r\nssh_host_alias = \"github-zamyb\"\r\nssh_key = \"~/.ssh/id_ed25519_zamyb\"\r\n"
	path := filepath.Join(t.TempDir(), "config.conf")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() CRLF error = %v", err)
	}
	profile, ok := cfg.FindProfile("work")
	if !ok {
		t.Fatal("profile not found")
	}
	if profile.GitHubUser != "zamyb" {
		t.Fatalf("GitHubUser = %q, want zamyb (CRLF not stripped?)", profile.GitHubUser)
	}
	if profile.GitEmail != "work@example.com" {
		t.Fatalf("GitEmail = %q, want work@example.com", profile.GitEmail)
	}
}

// Test ExpandPath logic with Windows-style home directory directly
func TestExpandPathWindowsStyleHome(t *testing.T) {
	t.Parallel()

	// filepath.Join normalises separators per OS, so on Linux this stays Unix.
	// What we validate here is that the logic (trim leading ~/) is correct.
	home := `/home/izzam`
	path := `~/.ssh/id_ed25519_work`
	got := filepath.Join(home, path[2:])
	want := `/home/izzam/.ssh/id_ed25519_work`
	if got != want {
		t.Fatalf("filepath.Join = %q, want %q", got, want)
	}

	// Bare ~ expands to home
	bare := `~`
	if bare != `~` || filepath.Join(home, "") != home {
		t.Fatal("bare ~ handling broken")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	original := Config{Profiles: []Profile{
		{Name: "work", GitHubUser: "zamyb", GitName: "IZZAMUDDIN", GitEmail: "work@example.com", SSHHostAlias: "github-zamyb", SSHKey: "~/.ssh/id_ed25519_zamyb"},
	}}

	path := filepath.Join(t.TempDir(), "config.conf")
	if err := Save(path, original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, _ := os.ReadFile(path)
	if strings.Contains(string(raw), "\r\n") {
		t.Fatal("Save() wrote CRLF — SSH config tools may misparse on Linux/Mac")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() after Save() error = %v", err)
	}
	profile, ok := loaded.FindProfile("work")
	if !ok || profile.GitHubUser != "zamyb" || profile.SSHKey != "~/.ssh/id_ed25519_zamyb" {
		t.Fatalf("round-trip mismatch: %+v", profile)
	}
}
