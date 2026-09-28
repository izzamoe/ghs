package e2e

import (
	"slices"
	"strings"
)

// ssh implements the "Fake ssh" table of contracts/fake-tools.md. Only the
// exact non-mutating argument vector used by ghs doctor is accepted.
func (f *fakeCtx) ssh(args []string) int {
	prefix := []string{"-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10"}
	if len(args) != len(prefix)+1 || !slices.Equal(args[:len(prefix)], prefix) {
		return f.unsupported(args)
	}
	alias, ok := strings.CutPrefix(args[len(prefix)], "git@")
	if !ok || alias == "" {
		return f.unsupported(args)
	}
	entry, known := f.st.SSH.Aliases[alias]
	switch {
	case !known:
		return f.failf(255, "ssh: Could not resolve hostname %s: Name or service not known", alias)
	case entry.Login != "":
		return f.failf(1, "Hi %s! You've successfully authenticated, but GitHub does not provide shell access.", entry.Login)
	case entry.Result == "denied":
		return f.failf(255, "git@github.com: Permission denied (publickey).")
	case entry.Result == "hostkey":
		return f.failf(255, "Host key verification failed.")
	case entry.Result == "timeout":
		return f.failf(255, "ssh: connect to host github.com port 22: Connection timed out")
	}
	return f.failf(255, "ssh: Could not resolve hostname %s: Name or service not known", alias)
}
