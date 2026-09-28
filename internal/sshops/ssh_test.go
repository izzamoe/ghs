package sshops

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestHasHostBlock(t *testing.T) {
	t.Parallel()

	content := `Host github-work
  HostName github.com

Host github-personal github-alt
  HostName github.com
`
	if !hasHostBlock(content, "github-work") {
		t.Fatal("hasHostBlock() did not detect single host")
	}
	if !hasHostBlock(content, "github-alt") {
		t.Fatal("hasHostBlock() did not detect multi host alias")
	}
	if !hasHostBlock(content, "GitHub-Work") {
		t.Fatal("hasHostBlock() must match case-insensitively")
	}
	if hasHostBlock(content, "github-missing") {
		t.Fatal("hasHostBlock() detected missing host")
	}
	if !hasHostBlock("host=github-eq\n", "github-eq") {
		t.Fatal("hasHostBlock() did not accept Host=alias")
	}
}

type recordingRunner struct{ calls []string }

func (r *recordingRunner) Run(name string, args ...string) error {
	r.calls = append(r.calls, name+" "+strings.Join(args, " "))
	return nil
}

func TestEnsureConfigQuotesPathWithWhitespace(t *testing.T) {
	t.Parallel()

	cfg := filepath.Join(t.TempDir(), ".ssh", "config")
	added, err := EnsureConfigAt(cfg, "github-work", "/home/my user/.ssh/id_ed25519_work")
	if err != nil || !added {
		t.Fatalf("EnsureConfigAt() = %v, %v", added, err)
	}
	data, _ := os.ReadFile(cfg)
	want := "Host github-work\n  HostName github.com\n  User git\n  IdentityFile \"/home/my user/.ssh/id_ed25519_work\"\n  IdentitiesOnly yes\n"
	if string(data) != want {
		t.Fatalf("config:\n%q\nwant:\n%q", data, want)
	}
}

func TestEnsureConfigUnquotedWithoutWhitespace(t *testing.T) {
	t.Parallel()

	cfg := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(cfg, []byte("Host other\n  HostName example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureConfigAt(cfg, "github-work", `C:\Users\x\.ssh\id_ed25519_work`); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(cfg)
	want := "Host other\n  HostName example.com\n\nHost github-work\n  HostName github.com\n  User git\n  IdentityFile " + filepath.ToSlash(`C:\Users\x\.ssh\id_ed25519_work`) + "\n  IdentitiesOnly yes\n"
	if string(data) != want {
		t.Fatalf("config:\n%q\nwant:\n%q", data, want)
	}
}

func TestEnsureConfigLeavesFileUntouchedWhenHostPresent(t *testing.T) {
	t.Parallel()

	cfg := filepath.Join(t.TempDir(), "config")
	original := "Host a github-WORK b\n  HostName github.com"
	if err := os.WriteFile(cfg, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(cfg, old, old); err != nil {
		t.Fatal(err)
	}
	added, err := EnsureConfigAt(cfg, "github-work", "/k")
	if err != nil || added {
		t.Fatalf("EnsureConfigAt() = %v, %v; want untouched", added, err)
	}
	data, _ := os.ReadFile(cfg)
	info, _ := os.Stat(cfg)
	if string(data) != original || !info.ModTime().Equal(old) {
		t.Fatalf("file modified: %q mtime %v", data, info.ModTime())
	}
}

func TestEnsureConfigStartsBlockOnNewLine(t *testing.T) {
	t.Parallel()

	cfg := filepath.Join(t.TempDir(), "config")
	original := "Host other\n  HostName example.com" // no trailing newline
	if err := os.WriteFile(cfg, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureConfigAt(cfg, "github-work", "/k"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(cfg)
	s := string(data)
	if !strings.HasPrefix(s, original+"\n") {
		t.Fatalf("existing content not preserved as a prefix: %q", s)
	}
	if !strings.Contains(s, "\nHost github-work\n") {
		t.Fatalf("block does not start on its own line: %q", s)
	}
}

func TestEnsureConfigFailsWhenUnreadable(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("mode 000 does not deny reads here")
	}

	cfg := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(cfg, []byte("Host x\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureConfigAt(cfg, "github-work", "/k"); err == nil {
		t.Fatal("EnsureConfigAt() error = nil for unreadable config")
	}
	_ = os.Chmod(cfg, 0o600)
	if data, _ := os.ReadFile(cfg); string(data) != "Host x\n" {
		t.Fatalf("config modified: %q", data)
	}
}

func TestEnsureKeyFailsWhenPublicKeyMissing(t *testing.T) {
	t.Parallel()

	key := filepath.Join(t.TempDir(), "id_ed25519_work")
	if err := os.WriteFile(key, []byte("PRIVATE"), 0o600); err != nil {
		t.Fatal(err)
	}
	run := &recordingRunner{}
	created, err := New(run).EnsureKeyAt(key, "zamyb-work")
	if err == nil || created || !strings.Contains(err.Error(), ".pub") {
		t.Fatalf("EnsureKeyAt() = %v, %v; want error naming the .pub file", created, err)
	}
	if len(run.calls) != 0 {
		t.Fatalf("ssh-keygen called: %v", run.calls)
	}
	if data, _ := os.ReadFile(key); string(data) != "PRIVATE" {
		t.Fatal("private key modified")
	}

	// Both present: nothing to do. Neither present: one ssh-keygen call.
	if err := os.WriteFile(key+".pub", []byte("ssh-ed25519 AAAA x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if created, err := New(run).EnsureKeyAt(key, "c"); err != nil || created {
		t.Fatalf("EnsureKeyAt(existing) = %v, %v", created, err)
	}
	fresh := filepath.Join(t.TempDir(), "sub", "id_new")
	if created, err := New(run).EnsureKeyAt(fresh, "c"); err != nil || !created {
		t.Fatalf("EnsureKeyAt(new) = %v, %v", created, err)
	}
	if len(run.calls) != 1 || run.calls[0] != "ssh-keygen -t ed25519 -C c -f "+fresh+" -N " {
		t.Fatalf("calls = %q", run.calls)
	}
}

func TestQuoteIfNeeded(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"/a/b":             "/a/b",
		"/a b/c":           `"/a b/c"`,
		"C:/Users/x y/key": `"C:/Users/x y/key"`,
		"/tab\there":       "\"/tab\there\"",
	} {
		if got := quoteIfNeeded(in); got != want {
			t.Errorf("quoteIfNeeded(%q) = %q, want %q", in, got, want)
		}
	}
}
