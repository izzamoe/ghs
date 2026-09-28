package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
	if profile.Workspace != "~/Documents/work" {
		t.Fatalf("Workspace = %q", profile.Workspace)
	}
}

func loadString(t *testing.T, content string) (Config, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.conf")
	if err := osWriteFile(path, content); err != nil {
		t.Fatal(err)
	}
	return Load(path)
}

func TestLoadRejectsDuplicateSections(t *testing.T) {
	t.Parallel()

	_, err := loadString(t, "[work]\ngh_user = \"a\"\n\n[Work]\ngh_user = \"b\"\n")
	if err == nil {
		t.Fatal("Load() error = nil, want duplicate error")
	}
	for _, want := range []string{"line 4", "line 1", "duplicate"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not mention %q", err, want)
		}
	}
}

func TestLoadRejectsKeyOutsideSection(t *testing.T) {
	t.Parallel()

	_, err := loadString(t, "# comment\ngh_user = \"a\"\n[work]\n")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("Load() error = %v, want line 2", err)
	}
}

func TestLoadRejectsUnterminatedQuote(t *testing.T) {
	t.Parallel()

	_, err := loadString(t, "[work]\ngh_user = \"zamyb\ngit_name = \"x\"\n")
	if err == nil || !strings.Contains(err.Error(), "line 2") || !strings.Contains(err.Error(), "unterminated quote") {
		t.Fatalf("Load() error = %v, want unterminated quote on line 2", err)
	}
}

func TestLoadRejectsControlCharacters(t *testing.T) {
	t.Parallel()

	for _, content := range []string{
		"[work]\ngit_name = \"a\x01b\"\n",
		"[work]\ngit_name = \"a\tb\"\n",
		"[wo\x7frk]\n",
	} {
		_, err := loadString(t, content)
		if err == nil || !strings.Contains(err.Error(), "control character") {
			t.Fatalf("Load(%q) error = %v, want control character", content, err)
		}
	}
	// Invalid lines are reported with their number too.
	_, err := loadString(t, "[work]\n\njust words\n")
	if err == nil || !strings.Contains(err.Error(), "line 3") {
		t.Fatalf("Load() error = %v, want line 3", err)
	}
	_, err = loadString(t, "[work\n")
	if err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("Load() error = %v, want line 1", err)
	}
}

func TestLoadPreservesUnknownKeysInOrder(t *testing.T) {
	t.Parallel()

	cfg, err := loadString(t, "[work]\nfuture_b = \"2\"\ngh_user = \"zamyb\"\nfuture_a = 1\n")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := []KeyValue{{Key: "future_b", Value: "2"}, {Key: "future_a", Value: "1"}}
	if got := cfg.Profiles[0].Extra; !slices.Equal(got, want) {
		t.Fatalf("Extra = %v, want %v", got, want)
	}
}

func TestLoadMissingFileIsNotFound(t *testing.T) {
	t.Parallel()

	cfg, err := Load(filepath.Join(t.TempDir(), "absent.conf"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load() error = %v, want fs.ErrNotExist", err)
	}
	if len(cfg.Profiles) != 0 {
		t.Fatalf("profiles = %v, want none", cfg.Profiles)
	}
}

func TestLoadUnreadableIsNotNotFound(t *testing.T) {
	t.Parallel()

	// A directory in place of the file is an I/O error, not "missing".
	dir := t.TempDir()
	_, err := Load(dir)
	if err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load(dir) error = %v, want a non-not-found error", err)
	}
}

func TestLoadKeepsCRLF(t *testing.T) {
	t.Parallel()

	cfg, err := loadString(t, "[work]\r\ngh_user = \"zamyb\"\r\nfuture = \"x\"\r\n")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	p := cfg.Profiles[0]
	if p.GitHubUser != "zamyb" || p.Extra[0].Value != "x" {
		t.Fatalf("profile = %+v", p)
	}
}

func TestFindProfileHelpers(t *testing.T) {
	t.Parallel()

	cfg := Config{Profiles: []Profile{
		{Name: "Work", GitHubUser: "Zamyb-Work", GitEmail: "Work@Example.com", SSHHostAlias: "github-work", Workspace: "~/Documents/work"},
		{Name: "me", GitHubUser: "zamyb", GitEmail: "me@example.com", SSHHostAlias: "github-me"},
	}}
	if _, ok := cfg.FindProfile("work"); ok {
		t.Fatal("FindProfile must be exact")
	}
	if p, ok := cfg.FindProfileFold("work"); !ok || p.Name != "Work" {
		t.Fatalf("FindProfileFold = %+v, %v", p, ok)
	}
	if p, ok := cfg.ByLogin("zamyb-work"); !ok || p.Name != "Work" {
		t.Fatalf("ByLogin = %+v, %v", p, ok)
	}
	if p, ok := cfg.ByEmail("work@example.com"); !ok || p.Name != "Work" {
		t.Fatalf("ByEmail = %+v, %v", p, ok)
	}
	if p, ok := cfg.ByAlias("GITHUB-ME"); !ok || p.Name != "me" {
		t.Fatalf("ByAlias = %+v, %v", p, ok)
	}
	if p, ok := cfg.ByWorkspace("~/Documents/work/"); !ok || p.Name != "Work" {
		t.Fatalf("ByWorkspace = %+v, %v", p, ok)
	}
	if _, ok := cfg.ByLogin(""); ok {
		t.Fatal("ByLogin(\"\") must not match")
	}
}

func osWriteFile(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}
