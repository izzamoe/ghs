package sshops

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseHostBlock(t *testing.T) {
	t.Parallel()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	content := `# global defaults
Host *
  ServerAliveInterval 30

Host a github-work b
  hostname github.com
  USER git
  IdentityFile "/p/with space/id"
  identitiesonly yes

Host github-me
  HostName=github.com
  User=git
  IdentityFile=~/.ssh/id_me
  IdentitiesOnly=yes

Match host foo
  User nobody

Host github-partial
  HostName ssh.github.com
Host github-next
  User git
`
	work := ParseHostBlock(content, "GitHub-Work")
	if !work.Found || work.HostName != "github.com" || work.User != "git" ||
		work.IdentityFile != "/p/with space/id" || work.IdentitiesOnly != "yes" {
		t.Fatalf("github-work block = %+v", work)
	}
	me := ParseHostBlock(content, "github-me")
	if !me.Found || me.HostName != "github.com" || me.User != "git" || me.IdentitiesOnly != "yes" ||
		me.IdentityFile != filepath.Join(home, ".ssh", "id_me") {
		t.Fatalf("github-me block = %+v", me)
	}
	partial := ParseHostBlock(content, "github-partial")
	if !partial.Found || partial.HostName != "ssh.github.com" || partial.User != "" {
		t.Fatalf("github-partial must stop at the next Host line: %+v", partial)
	}
	if b := ParseHostBlock(content, "github-missing"); b.Found {
		t.Fatalf("missing alias found: %+v", b)
	}
	if b := ParseHostBlock("Host x\n  HostName github.com\nMatch all\n  User git\n", "x"); b.User != "" {
		t.Fatalf("block must stop at Match: %+v", b)
	}
}

func TestHasHostBlockInConfig(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if ok, err := HasHostBlockInConfig(home, "github-work"); err != nil || ok {
		t.Fatalf("no file = %v, %v", ok, err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte("Host github-work\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ok, err := HasHostBlockInConfig(home, "github-work"); err != nil || !ok {
		t.Fatalf("present = %v, %v", ok, err)
	}
}
