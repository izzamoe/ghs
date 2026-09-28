package e2e

import (
	"strings"
	"testing"
)

func switchTo(login string) call {
	return mk("gh", "auth", "switch", "--hostname", "github.com", "--user", login)
}

func TestUse_NotInRepoFailsBeforeSwitch(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) { s.Git.Repo = false })

	res := sb.run("use", "me")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "not inside a git repository; pass --global to set the global identity")
	if n := sb.countCalls("gh", "auth switch"); n != 0 {
		t.Fatalf("%d account switches before failing", n)
	}
	sb.assertNoMutations()
}

func TestUse_IdentityFailureRestoresAccount(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) { s.Git.Fail = []string{"config user.email"} })

	res := sb.run("use", "me")
	assertCode(t, res, 1)
	sb.assertSequence(
		switchTo("zamyb"),
		mk("git", "config", "user.name", "Izzam"),
		mk("git", "config", "user.email", "me@example.com"),
		switchTo("zamyb-work"),
	)
	assertContains(t, res.Stderr, "forced failure (config user.email)")
	assertContains(t, res.Stderr, "restored gh account zamyb-work")
	if got := activeLogin(sb); got != "zamyb-work" {
		t.Fatalf("active account = %s, want zamyb-work restored", got)
	}
	if res.Stdout != "" {
		t.Fatalf("stdout on failure = %q, want empty", res.Stdout)
	}
}

func TestUse_NoEmailSwitchesOnlyAccount(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	sb.seedAccounts("zamyb-work", "zamyb-work", "olduser")

	res := sb.run("use", "old")
	assertCode(t, res, 0)
	want := "switched gh account: zamyb-work -> olduser\n" +
		"git identity unchanged: profile \"old\" has no email; run: ghs set-email old <email>\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}
	if n := sb.countCalls("git", "config"); n != 0 {
		t.Fatalf("%d git config calls, want none", n)
	}
}

func TestUse_UnauthenticatedFailsBeforeAnything(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard() // olduser is not logged in

	res := sb.run("use", "old")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "gh account olduser is not logged in")
	sb.assertNoMutations()
}

func TestUse_ReportsAccountAndLocalScope(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()

	res := sb.run("use", "me")
	assertCode(t, res, 0)
	want := "switched gh account: zamyb-work -> zamyb\n" +
		"git identity set: Izzam <me@example.com> (local: " + sb.repoDir + ")\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}
	st := sb.state()
	if st.Git.Local["user.email"][0] != "me@example.com" || st.Git.Local["user.name"][0] != "Izzam" {
		t.Fatalf("local identity = %v", st.Git.Local)
	}
}

func TestUse_ReportsGlobalScope(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) { s.Git.Repo = false })

	res := sb.run("use", "me", "--global")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "git identity set: Izzam <me@example.com> (global)\n")
	if got := sb.state().Git.Global["user.email"]; len(got) != 1 || got[0] != "me@example.com" {
		t.Fatalf("global email = %v", got)
	}
}

func TestUse_GlobalInsideRepoOnlyGlobal(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()

	res := sb.run("use", "me", "--global")
	assertCode(t, res, 0)
	for _, c := range sb.callsOf("git") {
		if isMutating(c) && !strings.Contains(strings.Join(c.Args, " "), "--global") {
			t.Fatalf("non-global git write %s", c)
		}
	}
	if len(sb.state().Git.Local) != 0 {
		t.Fatalf("local identity written: %v", sb.state().Git.Local)
	}
}

func TestUse_AlreadyActiveSkipsSwitch(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()

	res := sb.run("use", "work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "gh account already active: zamyb-work\n")
	if n := sb.countCalls("gh", "auth switch"); n != 0 {
		t.Fatalf("%d switches, want none", n)
	}
}
