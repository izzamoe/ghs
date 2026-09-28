package e2e

import (
	"os"
	"testing"
)

// statusSandbox: standard profiles, zamyb-work active, a repository whose
// local identity is the work identity and whose origin uses the work alias.
func statusSandbox(t *testing.T) *sandbox {
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) {
		s.Git.Origin = "git@github-work:acme/app.git"
		s.Git.Local = map[string]gitValues{"user.name": {"Izzam"}, "user.email": {"work@example.com"}}
	})
	return sb
}

const statusHead = "ghs status\n----------\n"

func TestStatus_AllResolveToSameProfileNoMismatch(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	res := sb.run("status")
	assertCode(t, res, 0)
	want := statusHead +
		"gh account:    zamyb-work (profile \"work\")\n" +
		"git identity:  Izzam <work@example.com> (profile \"work\", local)\n" +
		"origin:        git@github-work:acme/app.git (profile \"work\")\n"
	if res.Stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", res.Stdout, want)
	}
}

func TestStatus_OriginOtherProfileTwoMismatchLines(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.setState(func(s *fakeState) { s.Git.Origin = "git@github-me:acme/app.git" })
	res := sb.run("status")
	assertCode(t, res, 0)
	want := statusHead +
		"gh account:    zamyb-work (profile \"work\")\n" +
		"git identity:  Izzam <work@example.com> (profile \"work\", local)\n" +
		"origin:        git@github-me:acme/app.git (profile \"me\")\n" +
		"mismatch: origin resolves to profile \"me\" but gh account resolves to profile \"work\"\n" +
		"mismatch: origin resolves to profile \"me\" but git identity resolves to profile \"work\"\n"
	if res.Stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", res.Stdout, want)
	}
}

func TestStatus_IdentityNoProfile(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.setState(func(s *fakeState) { s.Git.Local["user.email"] = gitValues{"other@example.com"} })
	res := sb.run("status")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "git identity:  Izzam <other@example.com> (no profile, local)\n")
	assertContains(t, res.Stdout, "mismatch: git identity resolves to no profile (other@example.com)\n")
	assertNotContains(t, res.Stdout, "but git identity")
}

func TestStatus_AccountNoProfile(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.seedAccounts("stranger", "stranger")
	res := sb.run("status")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "gh account:    stranger (no profile)\n")
	assertContains(t, res.Stdout, "mismatch: gh account resolves to no profile (login stranger)\n")
}

func TestStatus_OriginGitHubDirectSuggestsFixRemote(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.setState(func(s *fakeState) { s.Git.Origin = "https://github.com/acme/app.git" })
	res := sb.run("status")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "origin:        https://github.com/acme/app.git (github.com, no alias)\n")
	assertContains(t, res.Stdout, "mismatch: origin uses github.com directly; run: ghs fix-remote work\n")

	sb.seedAccounts("stranger", "stranger")
	res = sb.run("status")
	assertContains(t, res.Stdout, "mismatch: origin uses github.com directly; run: ghs fix-remote <profile>\n")
}

func TestStatus_OutsideRepoGlobalIdentityOriginUnavailable(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.setState(func(s *fakeState) {
		s.Git.Repo = false
		s.Git.Global = map[string]gitValues{"user.name": {"Izzam"}, "user.email": {"me@example.com"}}
	})
	res := sb.run("status")
	assertCode(t, res, 0)
	want := statusHead +
		"gh account:    zamyb-work (profile \"work\")\n" +
		"git identity:  Izzam <me@example.com> (profile \"me\", global)\n" +
		"origin:        unavailable (not inside a git repository)\n" +
		"mismatch: git identity resolves to profile \"me\" but gh account resolves to profile \"work\"\n"
	if res.Stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", res.Stdout, want)
	}
	sb.assertSequence(mk("git", "config", "--global", "--show-origin", "--show-scope", "--get", "user.email"))
}

func TestStatus_NotGitHubOrigin(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.setState(func(s *fakeState) { s.Git.Origin = "https://gitlab.com/group/app.git" })
	res := sb.run("status")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "origin:        https://gitlab.com/group/app.git (not github: gitlab.com)\n")

	sb.setState(func(s *fakeState) { s.Git.Origin = "git@some-alias:group/app.git" })
	res = sb.run("status")
	assertContains(t, res.Stdout, "origin:        git@some-alias:group/app.git (unknown alias some-alias)\n")

	sb.setState(func(s *fakeState) { s.Git.Origin = "" })
	res = sb.run("status")
	assertContains(t, res.Stdout, "origin:        unavailable (no origin remote)\n")
}

func TestStatus_ScopeWordAndWorkspaceWord(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	repo := sb.expand("~/Documents/work/app")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	sb.setState(func(s *fakeState) {
		s.Git.Toplevel = repo
		s.Git.Origin = "git@github-work:acme/app.git"
		s.Git.Global["user.name"] = gitValues{"Izzam"}
		s.Git.Global["user.email"] = gitValues{"me@example.com"}
	})
	res := sb.runIn(repo, "status")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "git identity:  Izzam <work@example.com> (profile \"work\", global, workspace)\n")
	assertNotContains(t, res.Stdout, "mismatch")

	// Outside the workspace the plain global identity applies.
	res = sb.run("status")
	assertContains(t, res.Stdout, "git identity:  Izzam <me@example.com> (profile \"me\", global)\n")
}

func TestStatus_WithoutGH(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.without("gh")
	res := sb.run("status")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "gh account:    unavailable (gh not found)\n")
	assertContains(t, res.Stdout, "git identity:  Izzam <work@example.com> (profile \"work\", local)\n")
	assertNotContains(t, res.Stdout, "mismatch")
}

func TestStatus_NoMutations(t *testing.T) {
	t.Parallel()
	sb := statusSandbox(t)
	sb.setState(func(s *fakeState) { s.Git.Origin = "git@github.com:acme/app.git" })
	assertCode(t, sb.run("status"), 0)
	sb.assertNoMutations()
}
