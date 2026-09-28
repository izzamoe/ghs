package cli

import (
	"bytes"
	"strings"
	"testing"
)

func helpOutput(t *testing.T) string {
	t.Helper()
	var out, errOut bytes.Buffer
	if err := New(&out, &errOut).Run([]string{"--help"}); err != nil {
		t.Fatalf("help: %v", err)
	}
	return out.String()
}

func TestHelpStatesGitHubOnly(t *testing.T) {
	t.Parallel()
	help := helpOutput(t)
	for _, want := range []string{"Only github.com is supported", "Do not run ghs with sudo"} {
		if !strings.Contains(help, want) {
			t.Fatalf("help does not contain %q:\n%s", want, help)
		}
	}
}
