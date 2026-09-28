package e2e

import (
	"strings"
	"testing"
)

func TestSmoke_VersionRunsInSandbox(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	res := sb.run("version")
	assertCode(t, res, 0)
	if !strings.HasPrefix(res.Stdout, "ghs ") {
		t.Fatalf("stdout = %q, want prefix %q", res.Stdout, "ghs ")
	}
	if len(sb.calls()) != 0 {
		t.Fatalf("version invoked tools:\n%s", sb.logDump())
	}
}
