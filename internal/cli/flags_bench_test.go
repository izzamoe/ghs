package cli

import "testing"

var sinkPos []string
var sinkMap map[string]string

func BenchmarkParseArgs_Typical(b *testing.B) {
	spec, _ := lookupCommand("add-profile")
	args := []string{"work", "--gh-user", "zamyb", "--git-name", "IZZAMUDDIN", "--git-email", "work@example.com", "--ssh-alias", "github-work", "--ssh-key", "~/.ssh/id_ed25519_work"}
	for b.Loop() {
		sinkPos, sinkMap, _ = parseArgs(spec, args)
	}
}

func BenchmarkParseArgs_BoolFlags(b *testing.B) {
	spec, _ := lookupCommand("import-all")
	args := []string{"--require-email", "--no-overwrite"}
	for b.Loop() {
		sinkPos, sinkMap, _ = parseArgs(spec, args)
	}
}
