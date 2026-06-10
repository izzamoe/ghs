package sshops

import "testing"

var sinkBool bool

func BenchmarkHasHostBlock_Found(b *testing.B) {
	content := "Host github-work\n  HostName github.com\n  User git\n\nHost github-personal\n  HostName github.com\n"
	for b.Loop() {
		sinkBool = hasHostBlock(content, "github-work")
	}
}

func BenchmarkHasHostBlock_NotFound(b *testing.B) {
	content := "Host github-work\n  HostName github.com\n  User git\n\nHost github-personal\n  HostName github.com\n"
	for b.Loop() {
		sinkBool = hasHostBlock(content, "github-missing")
	}
}

func BenchmarkHasHostBlock_LargeConfig(b *testing.B) {
	content := ""
	for range 20 {
		content += "Host github-profile\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/key\n  IdentitiesOnly yes\n\n"
	}
	content += "Host github-target\n  HostName github.com\n"
	for b.Loop() {
		sinkBool = hasHostBlock(content, "github-target")
	}
}
