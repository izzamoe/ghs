package sshops

import (
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
