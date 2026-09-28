package sshops

import (
	"regexp"
	"strings"
)

// AuthKind classifies the outcome of `ssh -T git@<alias>`.
type AuthKind int

const (
	AuthUnknown AuthKind = iota
	AuthLogin
	AuthDenied
	AuthHostKey
	AuthUnresolvable
	AuthTimeout
)

// AuthResult is a classified ssh -T outcome. Message is the first line of
// the client's output.
type AuthResult struct {
	Kind    AuthKind
	Login   string
	Message string
}

var greeting = regexp.MustCompile(`Hi ([^!\s]+)! You've successfully authenticated`)

// ClassifySSHOutput reads GitHub's greeting or the SSH client's error. The
// exit status is ignored: GitHub closes an authenticated session with 1.
func ClassifySSHOutput(out string) AuthResult {
	res := AuthResult{Message: "no output from ssh"}
	for line := range strings.SplitSeq(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			res.Message = line
			break
		}
	}
	switch m := greeting.FindStringSubmatch(out); {
	case m != nil:
		res.Kind, res.Login = AuthLogin, m[1]
	case strings.Contains(out, "Permission denied"):
		res.Kind = AuthDenied
	case strings.Contains(out, "Host key verification failed"):
		res.Kind = AuthHostKey
	case strings.Contains(out, "Could not resolve hostname"):
		res.Kind = AuthUnresolvable
	case strings.Contains(out, "timed out"):
		res.Kind = AuthTimeout
	}
	return res
}

type combinedRunner interface {
	CombinedOutput(name string, args ...string) (string, int, error)
}

// Verify runs exactly `ssh -T -o BatchMode=yes -o ConnectTimeout=10
// git@<alias>`. BatchMode forbids prompts; no host-key options are passed,
// so the check can never write known_hosts (FR-074).
func Verify(run combinedRunner, alias string) (AuthResult, error) {
	out, _, err := run.CombinedOutput("ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "git@"+alias)
	if err != nil {
		return AuthResult{}, err
	}
	return ClassifySSHOutput(out), nil
}
