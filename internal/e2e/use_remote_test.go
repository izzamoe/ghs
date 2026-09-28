package e2e

import (
	"strings"
	"testing"
)

// originSandbox has the standard profiles, zamyb (profile "me") active, and
// the repository's origin set to url.
func originSandbox(t *testing.T, url string) *sandbox {
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	sb.seedAccounts("zamyb", "zamyb-work", "zamyb")
	sb.setState(func(s *fakeState) { s.Git.Origin = url })
	return sb
}

func warningLines(stderr string) []string {
	var out []string
	for line := range strings.SplitSeq(stderr, "\n") {
		if strings.HasPrefix(line, "ghs: warning:") {
			out = append(out, line)
		}
	}
	return out
}

func TestUse_WarnsWhenOriginIsGitHub(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github.com:acme/app.git")

	res := sb.run("use", "work")
	assertCode(t, res, 0)
	want := `ghs: warning: origin git@github.com:acme/app.git still uses github.com; pushes will not use the key of profile "work"; run: ghs use work --fix-remote  or  ghs fix-remote work`
	if got := warningLines(res.Stderr); len(got) != 1 || got[0] != want {
		t.Fatalf("warnings = %q, want exactly [%q]", got, want)
	}
	if strings.TrimSpace(res.Stderr) != want {
		t.Fatalf("stderr has more than the warning: %q", res.Stderr)
	}
	if n := sb.countCalls("git", "set-url"); n != 0 {
		t.Fatalf("%d set-url calls, want none", n)
	}
}

func TestUse_WarnsWhenOriginIsOtherProfileAlias(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github-me:acme/app.git")

	res := sb.run("use", "work")
	assertCode(t, res, 0)
	want := `ghs: warning: origin git@github-me:acme/app.git uses the alias of profile "me"; run: ghs use work --fix-remote  or  ghs fix-remote work`
	if got := warningLines(res.Stderr); len(got) != 1 || got[0] != want {
		t.Fatalf("warnings = %q, want exactly [%q]", got, want)
	}
}

func TestUse_NoWarningWhenAliasCorrect(t *testing.T) {
	t.Parallel()
	for _, url := range []string{"git@github-work:acme/app.git", "ssh://git@GitHub-Work/acme/app.git"} {
		sb := originSandbox(t, url)
		res := sb.run("use", "work")
		assertCode(t, res, 0)
		if res.Stderr != "" {
			t.Fatalf("%s: stderr = %q, want empty", url, res.Stderr)
		}
	}
}

func TestUse_NoWarningWithoutOrigin(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "")
	res := sb.run("use", "work")
	assertCode(t, res, 0)
	if res.Stderr != "" {
		t.Fatalf("stderr = %q, want empty", res.Stderr)
	}
}

func TestUse_NoWarningOutsideRepoWithGlobal(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github.com:acme/app.git")
	sb.setState(func(s *fakeState) { s.Git.Repo = false })
	res := sb.run("use", "work", "--global")
	assertCode(t, res, 0)
	if res.Stderr != "" {
		t.Fatalf("stderr = %q, want empty", res.Stderr)
	}
}

func TestUse_FixRemoteRewritesThenSwitchesThenSetsIdentity(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github.com:acme/app.git")

	res := sb.run("use", "work", "--fix-remote")
	assertCode(t, res, 0)
	sb.assertSequence(
		mk("git", "remote", "set-url", "origin", "git@github-work:acme/app.git"),
		switchTo("zamyb-work"),
		mk("git", "config", "user.name", "Izzam"),
		mk("git", "config", "user.email", "work@example.com"),
	)
	want := "origin updated: git@github.com:acme/app.git -> git@github-work:acme/app.git\n" +
		"switched gh account: zamyb -> zamyb-work\n" +
		"git identity set: Izzam <work@example.com> (local: " + sb.repoDir + ")\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}
	if res.Stderr != "" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	if got := sb.state().Git.Origin; got != "git@github-work:acme/app.git" {
		t.Fatalf("origin = %s", got)
	}
}

func TestUse_FixRemoteRejectsNonGitHubBeforeMutation(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "https://gitlab.com/group/app.git")
	res := sb.run("use", "work", "--fix-remote")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "gitlab.com")
	sb.assertNoMutations()
}

func TestUse_FixRemoteRejectsUnknownAlias(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@some-alias:acme/app.git")
	res := sb.run("use", "work", "--fix-remote")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "some-alias")
	sb.assertNoMutations()
}

func TestUse_FixRemoteMissingOriginFails(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "")
	res := sb.run("use", "work", "--fix-remote")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "no origin remote")
	sb.assertNoMutations()
}

func TestUse_FixRemoteOutsideRepoFailsEvenWithGlobal(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github.com:acme/app.git")
	sb.setState(func(s *fakeState) { s.Git.Repo = false })
	res := sb.run("use", "work", "--fix-remote", "--global")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "not inside a git repository")
	sb.assertNoMutations()
}

func TestUse_FixRemoteAlreadyCorrectNoSetURL(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github-work:acme/app.git")
	res := sb.run("use", "work", "--fix-remote")
	assertCode(t, res, 0)
	if !strings.HasPrefix(res.Stdout, "origin already correct: git@github-work:acme/app.git\n") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if n := sb.countCalls("git", "set-url"); n != 0 {
		t.Fatalf("%d set-url calls, want none", n)
	}
}

func TestUse_FixRemoteRollbackRestoresOriginAndAccount(t *testing.T) {
	t.Parallel()
	sb := originSandbox(t, "git@github.com:acme/app.git")
	sb.setState(func(s *fakeState) { s.Git.Fail = []string{"config user.email"} })

	res := sb.run("use", "work", "--fix-remote")
	assertCode(t, res, 1)
	calls := sb.calls()
	tail := calls[len(calls)-2:]
	if tail[0].String() != "gh auth switch --hostname github.com --user zamyb" ||
		tail[1].String() != "git remote set-url origin git@github.com:acme/app.git" {
		t.Fatalf("log does not end with the reverse-order rollback:\n%s", sb.logDump())
	}
	for _, want := range []string{"forced failure (config user.email)", "restored gh account zamyb", "restored origin to git@github.com:acme/app.git"} {
		assertContains(t, res.Stderr, want)
	}
	if got := sb.state().Git.Origin; got != "git@github.com:acme/app.git" {
		t.Fatalf("origin = %s, want restored", got)
	}
	if got := activeLogin(sb); got != "zamyb" {
		t.Fatalf("active = %s, want zamyb", got)
	}
	if res.Stdout != "" {
		t.Fatalf("stdout = %q, want empty on failure", res.Stdout)
	}
}
