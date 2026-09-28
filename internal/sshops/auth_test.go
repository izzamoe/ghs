package sshops

import (
	"strings"
	"testing"
)

func TestClassifySSHOutput(t *testing.T) {
	t.Parallel()
	tests := []struct {
		out     string
		kind    AuthKind
		login   string
		message string
	}{
		{"Hi zamyb-work! You've successfully authenticated, but GitHub does not provide shell access.\n", AuthLogin, "zamyb-work", "Hi zamyb-work! You've successfully authenticated, but GitHub does not provide shell access."},
		{"Warning: Permanently added 'github.com' to the list of known hosts.\r\nHi zamyb! You've successfully authenticated, but GitHub does not provide shell access.\r\n", AuthLogin, "zamyb", "Warning: Permanently added 'github.com' to the list of known hosts."},
		{"git@github.com: Permission denied (publickey).\n", AuthDenied, "", "git@github.com: Permission denied (publickey)."},
		{"Host key verification failed.\n", AuthHostKey, "", "Host key verification failed."},
		{"ssh: Could not resolve hostname github-x: Name or service not known\n", AuthUnresolvable, "", "ssh: Could not resolve hostname github-x: Name or service not known"},
		{"ssh: connect to host github.com port 22: Connection timed out\n", AuthTimeout, "", "ssh: connect to host github.com port 22: Connection timed out"},
		{"\nsomething unexpected\nsecond line\n", AuthUnknown, "", "something unexpected"},
		{"", AuthUnknown, "", "no output from ssh"},
	}
	for _, tt := range tests {
		got := ClassifySSHOutput(tt.out)
		if got.Kind != tt.kind || got.Login != tt.login || got.Message != tt.message {
			t.Errorf("ClassifySSHOutput(%q) = %+v, want kind %v login %q message %q", tt.out, got, tt.kind, tt.login, tt.message)
		}
	}
}

type combinedStub struct {
	args []string
	out  string
	code int
}

func (s *combinedStub) CombinedOutput(name string, args ...string) (string, int, error) {
	s.args = append([]string{name}, args...)
	return s.out, s.code, nil
}

func TestVerifyRunsExactNonMutatingCommand(t *testing.T) {
	t.Parallel()
	stub := &combinedStub{out: "Hi zamyb-work! You've successfully authenticated, but GitHub does not provide shell access.", code: 1}
	res, err := Verify(stub, "github-work")
	if err != nil || res.Kind != AuthLogin || res.Login != "zamyb-work" {
		t.Fatalf("Verify() = %+v, %v", res, err)
	}
	want := "ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@github-work"
	if got := strings.Join(stub.args, " "); got != want {
		t.Fatalf("ran %q, want %q", got, want)
	}
	for _, forbidden := range []string{"StrictHostKeyChecking", "UserKnownHostsFile"} {
		if strings.Contains(strings.Join(stub.args, " "), forbidden) {
			t.Fatalf("Verify passed %s", forbidden)
		}
	}
}
