package e2e

import (
	"os"
	"testing"
)

// uploadSandbox has profile work (zamyb-work) with zamyb active.
func uploadSandbox(t *testing.T) *sandbox {
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	sb.seedAccounts("zamyb", "zamyb", "zamyb-work")
	return sb
}

func activeLogin(sb *sandbox) string {
	for _, a := range sb.state().GH.Accounts {
		if a.Active {
			return a.Login
		}
	}
	return ""
}

func TestInitSSHUpload_SwitchesUploadsRestores(t *testing.T) {
	t.Parallel()
	sb := uploadSandbox(t)
	key := sb.expand("~/.ssh/id_ed25519_work")

	res := sb.run("init-ssh", "work", "--upload")
	assertCode(t, res, 0)
	sb.assertSequence(
		mk("gh", "auth", "status", "--hostname", "github.com", "--json", "hosts"),
		mk("ssh-keygen", "-t", "ed25519", "-C", "zamyb-work", "-f", key, "-N", ""),
		mk("gh", "auth", "switch", "--hostname", "github.com", "--user", "zamyb-work"),
		mk("gh", "ssh-key", "add", key+".pub", "--title", "ghs-work"),
		mk("gh", "auth", "switch", "--hostname", "github.com", "--user", "zamyb"),
	)
	if got := activeLogin(sb); got != "zamyb" {
		t.Fatalf("active account = %s, want zamyb restored", got)
	}
	if keys := sb.state().GH.SSHKeys["zamyb-work"]; len(keys) != 1 {
		t.Fatalf("keys on zamyb-work = %v, want the uploaded key", keys)
	}
	if keys := sb.state().GH.SSHKeys["zamyb"]; len(keys) != 0 {
		t.Fatalf("key uploaded to the wrong account: %v", keys)
	}
	assertContains(t, res.Stdout, "uploaded public key "+key+".pub to account zamyb-work")
}

func TestInitSSHUpload_FailureRestoresAndReportsBoth(t *testing.T) {
	t.Parallel()
	sb := uploadSandbox(t)
	sb.setState(func(s *fakeState) { s.GH.Fail = []string{"ssh-key add"} })

	res := sb.run("init-ssh", "work", "--upload")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "forced failure (ssh-key add)")
	assertContains(t, res.Stderr, "restored gh account zamyb")
	if got := activeLogin(sb); got != "zamyb" {
		t.Fatalf("active account = %s, want zamyb restored", got)
	}
	calls := sb.callsOf("gh")
	if last := calls[len(calls)-1].String(); last != "gh auth switch --hostname github.com --user zamyb" {
		t.Fatalf("last gh call = %s", last)
	}
}

func TestInitSSHUpload_UnauthenticatedFailsBeforeKeygen(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	sb.seedAccounts("zamyb", "zamyb")

	res := sb.run("init-ssh", "work", "--upload")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "zamyb-work")
	assertContains(t, res.Stderr, "gh auth login")
	if n := len(sb.callsOf("ssh-keygen")); n != 0 {
		t.Fatalf("ssh-keygen ran %d times", n)
	}
	if sb.sshConfig() != "" {
		t.Fatalf("ssh config written:\n%s", sb.sshConfig())
	}
	sb.assertNoMutations()
}

func TestInitSSHUpload_NoSwitchWhenAlreadyActive(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")

	res := sb.run("init-ssh", "work", "--upload")
	assertCode(t, res, 0)
	if n := sb.countCalls("gh", "auth switch"); n != 0 {
		t.Fatalf("%d account switches, want none:\n%s", n, sb.logDump())
	}
	if keys := sb.state().GH.SSHKeys["zamyb-work"]; len(keys) != 1 {
		t.Fatalf("keys = %v", keys)
	}
}

func TestInitSSHUpload_AlreadyRegisteredIsSuccess(t *testing.T) {
	t.Parallel()
	sb := uploadSandbox(t)
	key := sb.touchKey("~/.ssh/id_ed25519_work")
	pub, _ := os.ReadFile(key + ".pub")
	sb.setState(func(s *fakeState) { s.GH.SSHKeys = map[string][]string{"zamyb-work": {string(pub)}} })

	res := sb.run("init-ssh", "work", "--upload")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "public key "+key+".pub already registered on account zamyb-work")
	if got := activeLogin(sb); got != "zamyb" {
		t.Fatalf("active account = %s, want zamyb", got)
	}
}

func TestCloneUploadKey_UploadsWhileSwitchedAndStaysSwitched(t *testing.T) {
	t.Parallel()
	sb := uploadSandbox(t)
	key := sb.expand("~/.ssh/id_ed25519_work")

	res := sb.run("clone", "work", "acme/app", "--upload-key")
	assertCode(t, res, 0)
	sb.assertSequence(
		mk("gh", "auth", "switch", "--hostname", "github.com", "--user", "zamyb-work"),
		mk("gh", "ssh-key", "add", key+".pub", "--title", "ghs-work"),
		mk("git", "clone", "git@github-work:acme/app.git", "app"),
		mk("git", "-C", "app", "config", "user.name", "Izzam"),
		mk("git", "-C", "app", "config", "user.email", "work@example.com"),
	)
	for _, c := range sb.callsOf("gh") {
		if c.String() == "gh auth switch --hostname github.com --user zamyb" {
			t.Fatalf("clone switched back to zamyb; it must stay on the profile's account:\n%s", sb.logDump())
		}
	}
	if got := activeLogin(sb); got != "zamyb-work" {
		t.Fatalf("active account = %s, want zamyb-work", got)
	}
	if keys := sb.state().GH.SSHKeys["zamyb-work"]; len(keys) != 1 {
		t.Fatalf("keys = %v", keys)
	}
	assertContains(t, res.Stdout, "switched gh account: zamyb -> zamyb-work")
}

func TestInitSSHUpload_UnreadablePubFailsBeforeSwitch(t *testing.T) {
	t.Parallel()
	skipUnlessPermissionsEnforced(t)
	sb := uploadSandbox(t)
	key := sb.touchKey("~/.ssh/id_ed25519_work")
	if err := os.Chmod(key+".pub", 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(key+".pub", 0o644) })

	res := sb.run("init-ssh", "work", "--upload")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, key+".pub")
	sb.assertNoMutations()
}
