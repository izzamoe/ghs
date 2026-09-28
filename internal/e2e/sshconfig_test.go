package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitSSH_SpaceInHomeQuotesIdentityFile(t *testing.T) {
	t.Parallel()
	sb := newSandboxHome(t, "my home")
	sb.writeConfig(cfgWork)

	res := sb.run("init-ssh", "work")
	assertCode(t, res, 0)
	key := filepath.ToSlash(sb.expand("~/.ssh/id_ed25519_work"))
	if got, want := sb.sshConfig(), sshBlock("github-work", `"`+key+`"`); got != want {
		t.Fatalf("ssh config:\n%s\nwant:\n%s", got, want)
	}
	assertContains(t, res.Stdout, `ssh is ready for profile "work" via host "github-work"`)
	sb.assertSequence(mk("ssh-keygen", "-t", "ed25519", "-C", "zamyb-work", "-f", sb.expand("~/.ssh/id_ed25519_work"), "-N", ""))
}

func TestInitSSH_SecondRunAddsNoBlock(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)

	assertCode(t, sb.run("init-ssh", "work"), 0)
	first := sb.sshConfig()
	res := sb.run("init-ssh", "work")
	assertCode(t, res, 0)
	if got := sb.sshConfig(); got != first {
		t.Fatalf("second run changed ssh config:\n%s", got)
	}
	if n := strings.Count(first, "Host github-work"); n != 1 {
		t.Fatalf("%d Host blocks, want 1:\n%s", n, first)
	}
	if n := len(sb.callsOf("ssh-keygen")); n != 1 {
		t.Fatalf("%d ssh-keygen calls, want 1", n)
	}
	assertNotContains(t, res.Stdout, "appended")
	assertNotContains(t, res.Stdout, "generated")
}

func TestInitSSH_UnreadableConfigNoKeygen(t *testing.T) {
	t.Parallel()
	skipUnlessPermissionsEnforced(t)
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	sb.writeSSHConfig("Host other\n")
	if err := os.Chmod(sb.sshCfgPath(), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sb.sshCfgPath(), 0o600) })

	res := sb.run("init-ssh", "work")
	assertCode(t, res, 1)
	if n := len(sb.callsOf("ssh-keygen")); n != 0 {
		t.Fatalf("ssh-keygen ran %d times before the unreadable config was detected", n)
	}
	if _, err := os.Stat(sb.expand("~/.ssh/id_ed25519_work")); !os.IsNotExist(err) {
		t.Fatalf("key created: %v", err)
	}
}

func TestInitSSH_PrivateWithoutPublicFails(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	key := sb.touchKey("~/.ssh/id_ed25519_work")
	if err := os.Remove(key + ".pub"); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(key)

	res := sb.run("init-ssh", "work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, ".pub")
	if n := len(sb.callsOf("ssh-keygen")); n != 0 {
		t.Fatalf("ssh-keygen called %d times", n)
	}
	if after, _ := os.ReadFile(key); string(after) != string(before) {
		t.Fatal("private key modified")
	}
	if sb.sshConfig() != "" {
		t.Fatalf("ssh config written:\n%s", sb.sshConfig())
	}
}
