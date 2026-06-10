package cli

import "testing"

var sinkMap map[string]string
var sinkBool bool

func BenchmarkParseFlags_Typical(b *testing.B) {
	args := []string{"--gh-user", "zamyb", "--git-name", "IZZAMUDDIN", "--git-email", "work@example.com", "--ssh-alias", "github-work", "--ssh-key", "~/.ssh/id_ed25519_work"}
	for b.Loop() {
		sinkMap = parseFlags(args)
	}
}

func BenchmarkParseFlags_BoolFlags(b *testing.B) {
	args := []string{"--require-email", "--no-overwrite"}
	for b.Loop() {
		sinkMap = parseFlags(args)
	}
}

func BenchmarkHasFlag(b *testing.B) {
	args := []string{"--global"}
	for b.Loop() {
		sinkBool = hasFlag(args, "--global")
	}
}
