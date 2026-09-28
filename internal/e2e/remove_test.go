package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func ghStateJSON(t *testing.T, sb *sandbox) string {
	t.Helper()
	data, err := json.Marshal(sb.state().GH)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRemove_MiddleProfilePreservesOthersAndExtraKeys(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	first := cfgWork + "future_key = \"kept\"\n"
	sb.writeConfig(first + "\n" + cfgMe + "\n" + cfgOld)

	res := sb.run("remove", "me")
	assertCode(t, res, 0)
	if got, want := sb.readConfig(), first+"\n"+cfgOld; got != want {
		t.Fatalf("config:\n%s\nwant:\n%s", got, want)
	}
	if !strings.HasPrefix(res.Stdout, "removed profile \"me\" from "+sb.cfgPath()+"\n") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestRemove_NotFoundExit1(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	res := sb.run("remove", "nosuch")
	assertCode(t, res, 1)
	if res.Stderr != "ghs: profile \"nosuch\" not found\n" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	if sb.readConfig() != cfgAll {
		t.Fatal("config changed")
	}
}

func TestRemove_NotFoundSuggestsCaseVariant(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	res := sb.run("remove", "Work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `profile "Work" not found; did you mean "work"?`)
	if sb.readConfig() != cfgAll {
		t.Fatal("config changed")
	}
}

func TestRemove_KeepsSSHFilesAndConfigByteIdentical(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()
	sb.touchKey("~/.ssh/id_ed25519_old")
	sb.touchKey("~/.ssh/id_ed25519_work")
	sb.writeSSHConfig(sshBlock("github-old", "~/.ssh/id_ed25519_old") + "\n" + sshBlock("github-work", "~/.ssh/id_ed25519_work"))
	sshBefore := sb.snapshot("~/.ssh")
	ghBefore := ghStateJSON(t, sb)

	for _, name := range []string{"old", "work"} {
		assertCode(t, sb.run("remove", name), 0)
	}
	assertSnapshotEqual(t, sshBefore, sb.snapshot("~/.ssh"))
	if got := ghStateJSON(t, sb); got != ghBefore {
		t.Fatalf("gh state changed:\n%s\n%s", ghBefore, got)
	}
	for _, c := range sb.callsOf("git") {
		if isMutating(c) {
			t.Fatalf("git write %s", c)
		}
	}
}

func TestRemove_NeverCallsLogoutOrSwitch(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = append(s.GH.Accounts, ghAccount{Login: "olduser", State: "success"})
	})
	assertCode(t, sb.run("remove", "old"), 0)
	calls := sb.callsOf("gh")
	if len(calls) != 1 || calls[0].String() != "gh auth status --hostname github.com --json hosts" {
		t.Fatalf("gh calls = %v", calls)
	}
}

func TestRemove_PrintsKeptLines(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = append(s.GH.Accounts, ghAccount{Login: "olduser", State: "success"})
	})
	sb.touchKey("~/.ssh/id_ed25519_old")
	sb.writeSSHConfig(sshBlock("github-old", "~/.ssh/id_ed25519_old"))

	res := sb.run("remove", "old")
	assertCode(t, res, 0)
	want := "removed profile \"old\" from " + sb.cfgPath() + "\n" +
		"kept: ssh key ~/.ssh/id_ed25519_old (delete by hand if unused)\n" +
		"kept: ssh config block \"Host github-old\" in ~/.ssh/config (edit by hand)\n" +
		"kept: gh account olduser is still logged in; run: gh auth logout --hostname github.com --user olduser\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}

	res = sb.run("remove", "work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "kept: gh account zamyb-work is still active; run: gh auth logout --hostname github.com --user zamyb-work\n")
	assertNotContains(t, res.Stdout, "kept: ssh key")
}

func TestRemove_UnlinksWorkspaceFirst(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	file := sb.identityFile("work")

	res := sb.run("remove", "work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "unlinked workspace ~/Documents/work (removed includeIf entry and "+file+")\n")
	sb.assertSequence(mk("git", "config", "--global", "--unset", "--fixed-value", wsKey, file))
	if len(globalValues(sb, wsKey)) != 0 {
		t.Fatal("include entry left")
	}
	if _, err := os.Stat(filepath.FromSlash(file)); !os.IsNotExist(err) {
		t.Fatal("identity file left")
	}
	if strings.Contains(sb.readConfig(), "[work]") {
		t.Fatal("profile not removed")
	}
}

func TestRemove_UnlinkFailureAbortsRemoval(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	before := sb.readConfig()
	sb.setState(func(s *fakeState) { s.Git.Fail = []string{"--unset"} })

	res := sb.run("remove", "work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "forced failure (--unset)")
	if sb.readConfig() != before {
		t.Fatal("config changed although unlinking failed")
	}
	if _, err := os.Stat(filepath.FromSlash(sb.identityFile("work"))); err != nil {
		t.Fatalf("identity file removed: %v", err)
	}
}

func TestRemove_LastProfileLeavesEmptyFileWithMode(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(sb.cfgPath(), 0o640); err != nil {
			t.Fatal(err)
		}
	}
	assertCode(t, sb.run("remove", "work"), 0)
	info, err := os.Stat(sb.cfgPath())
	if err != nil {
		t.Fatalf("config deleted: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("config size = %d, want 0", info.Size())
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640", info.Mode().Perm())
	}
	sb.without("gh")
	res := sb.run("list")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "no profiles found")
}

func TestRemove_UsageErrors(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"remove", "work", "extra"}, {"remove", "work", "--force"}, {"remove"}} {
		sb := newSandbox(t)
		sb.writeConfig(cfgAll)
		res := sb.run(args...)
		assertCode(t, res, 2)
		if sb.readConfig() != cfgAll || len(sb.calls()) != 0 {
			t.Fatalf("%v: side effects", args)
		}
	}
}

func TestRemove_WorksWithoutGH(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	sb.without("gh")
	res := sb.run("remove", "old")
	assertCode(t, res, 0)
	assertNotContains(t, res.Stdout, "gh account")
	if strings.Contains(sb.readConfig(), "[old]") {
		t.Fatal("profile not removed")
	}
}
