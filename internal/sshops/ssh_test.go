package sshops

import "testing"

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
	if hasHostBlock(content, "github-missing") {
		t.Fatal("hasHostBlock() detected missing host")
	}
}
