package sshops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Validate that IdentityFile path in SSH config uses forward slashes.
// On Windows, keyPath from ExpandPath uses backslashes; OpenSSH accepts both,
// but forward slashes are safer across all SSH client implementations.
func TestEnsureConfigIdentityFileUsesForwardSlash(t *testing.T) {
	t.Parallel()

	// Simulate a Unix key path — filepath.ToSlash is a no-op here (already /)
	unixKeyPath := `/home/izzam/.ssh/id_ed25519_work`
	block := buildSSHBlock("github-work", filepath.ToSlash(unixKeyPath))
	if !strings.Contains(block, "IdentityFile /home/izzam/.ssh/id_ed25519_work") {
		t.Fatalf("unexpected block:\n%s", block)
	}

	// Validate filepath.ToSlash converts backslash on any platform when given a slash path
	// (On Windows this would convert C:\... to C:/...)
	slashPath := filepath.ToSlash(`/home/izzam/.ssh/id_ed25519_work`)
	if strings.Contains(slashPath, `\`) {
		t.Fatalf("filepath.ToSlash left backslash in %q", slashPath)
	}
}

// Validate hasHostBlock handles CRLF SSH config (Windows-edited files)
func TestHasHostBlockCRLF(t *testing.T) {
	t.Parallel()

	content := "Host github-work\r\n  HostName github.com\r\n\r\nHost github-personal\r\n  HostName github.com\r\n"

	if !hasHostBlock(content, "github-work") {
		t.Fatal("hasHostBlock() failed to detect host in CRLF file")
	}
	if !hasHostBlock(content, "github-personal") {
		t.Fatal("hasHostBlock() failed to detect second host in CRLF file")
	}
	if hasHostBlock(content, "github-missing") {
		t.Fatal("hasHostBlock() false positive in CRLF file")
	}
}

func buildSSHBlock(alias, keyPath string) string {
	return "\nHost " + alias + "\n  HostName github.com\n  User git\n  IdentityFile " + keyPath + "\n  IdentitiesOnly yes\n"
}

// A Windows home such as C:\Users\John Doe produces key paths with spaces;
// the IdentityFile must then be quoted (and forward-slashed on Windows).
func TestEnsureConfigQuotesWindowsPathWithSpace(t *testing.T) {
	t.Parallel()

	key := `C:\Users\John Doe\.ssh\id_ed25519_work`
	cfg := filepath.Join(t.TempDir(), "config")
	if _, err := EnsureConfigAt(cfg, "github-work", key); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := `  IdentityFile "` + filepath.ToSlash(key) + `"` + "\n"
	if !strings.Contains(string(data), want) {
		t.Fatalf("config:\n%s\nwant line %q", data, want)
	}
	if block := ParseHostBlock(string(data), "github-work"); block.IdentityFile != filepath.ToSlash(key) {
		t.Fatalf("parsed IdentityFile = %q", block.IdentityFile)
	}
}

func TestParseHostBlockCRLF(t *testing.T) {
	t.Parallel()

	content := "Host github-work\r\n  HostName github.com\r\n  User git\r\n  IdentityFile \"C:/Users/x y/.ssh/id\"\r\n  IdentitiesOnly yes\r\n"
	block := ParseHostBlock(content, "github-work")
	if !block.Found || block.User != "git" || block.IdentityFile != "C:/Users/x y/.ssh/id" || block.IdentitiesOnly != "yes" {
		t.Fatalf("block = %+v", block)
	}
}
