package e2e

import (
	"testing"
)

func fixRemoteSandbox(t *testing.T, origin string) *sandbox {
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	sb.writeSSHConfig(sshBlock("github-work", "~/.ssh/id_ed25519_work"))
	sb.setState(func(s *fakeState) { s.Git.Origin = origin })
	return sb
}

func TestFixRemote_RejectsGitLab(t *testing.T) {
	t.Parallel()
	sb := fixRemoteSandbox(t, "git@gitlab.com:group/repo.git")
	res := sb.run("fix-remote", "work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "gitlab.com")
	if n := sb.countCalls("git", "set-url"); n != 0 {
		t.Fatalf("%d set-url calls", n)
	}
	if got := sb.state().Git.Origin; got != "git@gitlab.com:group/repo.git" {
		t.Fatalf("origin = %s", got)
	}
}

func TestFixRemote_RewritesOtherProfileAlias(t *testing.T) {
	t.Parallel()
	sb := fixRemoteSandbox(t, "git@github-me:owner/repo.git")
	res := sb.run("fix-remote", "work")
	assertCode(t, res, 0)
	sb.assertSequence(mk("git", "remote", "set-url", "origin", "git@github-work:owner/repo.git"))
	if res.Stdout != "origin updated: git@github-me:owner/repo.git -> git@github-work:owner/repo.git\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestFixRemote_RejectsUnknownAlias(t *testing.T) {
	t.Parallel()
	sb := fixRemoteSandbox(t, "git@some-alias:owner/repo.git")
	res := sb.run("fix-remote", "work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "some-alias")
	sb.assertNoMutations()
}

func TestFixRemote_AlreadyCorrectNoSetURL(t *testing.T) {
	t.Parallel()
	sb := fixRemoteSandbox(t, "git@github-work:owner/repo.git")
	res := sb.run("fix-remote", "work")
	assertCode(t, res, 0)
	if res.Stdout != "origin already correct: git@github-work:owner/repo.git\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	sb.assertNoMutations()
}

func TestFixRemote_HTTPSAndSSHSchemeForms(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ in, want string }{
		{"https://github.com/owner/repo.git", "git@github-work:owner/repo.git"},
		{"https://github.com/owner/repo", "git@github-work:owner/repo.git"},
		{"http://github.com/owner/repo.git", "git@github-work:owner/repo.git"},
		{"ssh://git@github.com/owner/repo.git", "git@github-work:owner/repo.git"},
		{"ssh://git@github.com:22/owner/repo", "git@github-work:owner/repo.git"},
		{"git@github.com:owner/repo.git", "git@github-work:owner/repo.git"},
		{"git@GitHub.com:owner/repo", "git@github-work:owner/repo.git"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			sb := fixRemoteSandbox(t, tc.in)
			res := sb.run("fix-remote", "work")
			assertCode(t, res, 0)
			if got := sb.state().Git.Origin; got != tc.want {
				t.Fatalf("origin = %s, want %s", got, tc.want)
			}
			assertContains(t, res.Stdout, "origin updated: "+tc.in+" -> "+tc.want)
		})
	}
}

func TestFixRemote_TrailingSlashAndGitSuffixNormalized(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"https://github.com/owner/repo/", "https://github.com/owner/repo.git/", "https://github.com/owner/repo.git.git"} {
		sb := fixRemoteSandbox(t, in)
		assertCode(t, sb.run("fix-remote", "work"), 0)
		if got := sb.state().Git.Origin; got != "git@github-work:owner/repo.git" {
			t.Fatalf("%s -> %s, want git@github-work:owner/repo.git", in, got)
		}
	}
}

func TestFixRemote_NoOriginFails(t *testing.T) {
	t.Parallel()
	sb := fixRemoteSandbox(t, "")
	res := sb.run("fix-remote", "work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "no origin remote")
	sb.assertNoMutations()
}

func TestFixRemote_WarnsWhenAliasMissingFromSSHConfig(t *testing.T) {
	t.Parallel()
	sb := fixRemoteSandbox(t, "git@github.com:owner/repo.git")
	sb.writeSSHConfig("Host other\n  HostName example.com\n")

	res := sb.run("fix-remote", "work")
	assertCode(t, res, 0)
	if res.Stderr != "ghs: warning: alias github-work has no block in ~/.ssh/config; run: ghs init-ssh work\n" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	if got := sb.state().Git.Origin; got != "git@github-work:owner/repo.git" {
		t.Fatalf("origin = %s", got)
	}

	// With the block present there is no warning.
	sb2 := fixRemoteSandbox(t, "git@github.com:owner/repo.git")
	res = sb2.run("fix-remote", "work")
	assertCode(t, res, 0)
	if res.Stderr != "" {
		t.Fatalf("stderr = %q, want empty", res.Stderr)
	}
}

func TestClone_RejectsNonGitHubBeforeMutation(t *testing.T) {
	t.Parallel()
	for _, url := range []string{"https://gitlab.com/group/repo.git", "git@gitlab.com:group/repo.git", "git@github-me:owner/repo.git", "https://ghe.example.com/o/r"} {
		sb := newSandbox(t)
		sb.writeConfig(cfgAll)
		sb.seedAccounts("zamyb", "zamyb", "zamyb-work")
		res := sb.run("clone", "work", url)
		assertCode(t, res, 1)
		if n := len(sb.calls()); n != 0 {
			t.Fatalf("%s: %d tool calls before rejecting:\n%s", url, n, sb.logDump())
		}
		if sb.sshConfig() != "" {
			t.Fatalf("%s: ssh config written", url)
		}
	}
}
