package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFakeGH_AuthStatusAndSwitch(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = []ghAccount{
			{Login: "zamyb-work", Active: true, State: "success"},
			{Login: "zamyb", State: "success"},
			{Login: "broken", State: "error"},
		}
	})

	res := sb.fake(sb.repoDir, "gh", "auth", "status", "--hostname", "github.com", "--json", "hosts")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, `"login":"zamyb-work"`)
	assertContains(t, res.Stdout, `"active":true`)

	assertCode(t, sb.fake(sb.repoDir, "gh", "auth", "switch", "--hostname", "github.com", "--user", "zamyb"), 0)
	st := sb.state()
	if st.GH.Accounts[0].Active || !st.GH.Accounts[1].Active {
		t.Fatalf("switch not persisted: %+v", st.GH.Accounts)
	}

	res = sb.fake(sb.repoDir, "gh", "auth", "switch", "--hostname", "github.com", "--user", "broken")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "could not switch")

	other := sb.fake(sb.repoDir, "gh", "auth", "status", "--hostname", "ghe.example.com", "--json", "hosts")
	assertCode(t, other, 0)
	assertContains(t, other.Stdout, `{"hosts":{}}`)

	sb.setState(func(s *fakeState) { s.GH.Accounts = nil })
	res = sb.fake(sb.repoDir, "gh", "auth", "status", "--hostname", "github.com", "--json", "hosts")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "You are not logged into any GitHub hosts")

	if got := len(sb.callsOf("gh")); got != 5 {
		t.Fatalf("logged %d gh calls, want 5", got)
	}
}

func TestFakeGH_SSHKeyAddAlreadyPresent(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	pub := filepath.Join(sb.root, "id.pub")
	if err := os.WriteFile(pub, []byte("ssh-ed25519 AAAAKEY comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = []ghAccount{{Login: "zamyb-work", Active: true, State: "success"}}
	})

	res := sb.fake(sb.repoDir, "gh", "ssh-key", "add", pub, "--title", "ghs-work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "Public key added")
	if keys := sb.state().GH.SSHKeys["zamyb-work"]; len(keys) != 1 {
		t.Fatalf("keys = %v, want one", keys)
	}

	res = sb.fake(sb.repoDir, "gh", "ssh-key", "add", pub, "--title", "ghs-work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "already exists")
	if keys := sb.state().GH.SSHKeys["zamyb-work"]; len(keys) != 1 {
		t.Fatalf("keys = %v, want still one", keys)
	}

	res = sb.fake(sb.repoDir, "gh", "ssh-key", "add", filepath.Join(sb.root, "missing.pub"), "--title", "x")
	assertCode(t, res, 1)

	res = sb.fake(sb.repoDir, "gh", "auth", "logout", "--hostname", "github.com", "--user", "zamyb-work")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "logout must never be called")
}

func TestFakeGit_ConfigShowOriginResolvesIncludeIf(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	workRepo := filepath.Join(sb.home, "Documents", "work", "app")
	if err := os.MkdirAll(workRepo, 0o700); err != nil {
		t.Fatal(err)
	}
	identity := filepath.Join(sb.root, "gitconfig-work")
	if err := os.WriteFile(identity, []byte("[user]\n\tname = Izzam\n\temail = work@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	sb.setState(func(s *fakeState) {
		s.Git = gitState{Repo: true, Toplevel: workRepo, Global: map[string]gitValues{
			"user.email": {"me@example.com"},
			"includeIf.gitdir/i:~/Documents/work/.path": {identity},
		}}
	})

	res := sb.fake(workRepo, "git", "config", "--show-origin", "--show-scope", "--get", "user.email")
	assertCode(t, res, 0)
	want := "global\tfile:" + filepath.ToSlash(identity) + "\twork@example.com\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}

	res = sb.fake(sb.repoDir, "git", "config", "--show-origin", "--show-scope", "--get", "user.email")
	assertCode(t, res, 0)
	want = "global\tfile:" + filepath.ToSlash(sb.home) + "/.gitconfig\tme@example.com\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}

	assertCode(t, sb.fake(workRepo, "git", "config", "user.email", "local@example.com"), 0)
	res = sb.fake(workRepo, "git", "config", "--show-origin", "--show-scope", "--get", "user.email")
	want = "local\tfile:" + filepath.ToSlash(workRepo) + "/.git/config\tlocal@example.com\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}

	res = sb.fake(workRepo, "git", "config", "--global", "--show-origin", "--show-scope", "--get", "user.email")
	assertContains(t, res.Stdout, "me@example.com")

	assertCode(t, sb.fake(workRepo, "git", "config", "--show-origin", "--show-scope", "--get", "user.signingkey"), 1)
}

func TestFakeGit_UnsetFixedValueExit5(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	key := "includeIf.gitdir/i:~/Documents/work/.path"
	assertCode(t, sb.fake(sb.repoDir, "git", "config", "--global", "--get-all", key), 1)
	assertCode(t, sb.fake(sb.repoDir, "git", "config", "--global", "--add", key, "/a"), 0)
	assertCode(t, sb.fake(sb.repoDir, "git", "config", "--global", "--add", key, "/b"), 0)
	res := sb.fake(sb.repoDir, "git", "config", "--global", "--get-all", key)
	if res.Stdout != "/a\n/b\n" {
		t.Fatalf("get-all = %q", res.Stdout)
	}
	assertCode(t, sb.fake(sb.repoDir, "git", "config", "--global", "--unset", "--fixed-value", key, "/a"), 0)
	assertCode(t, sb.fake(sb.repoDir, "git", "config", "--global", "--unset", "--fixed-value", key, "/a"), 5)
	if got := sb.state().Git.Global[key]; len(got) != 1 || got[0] != "/b" {
		t.Fatalf("remaining = %v, want [/b]", got)
	}
}

func TestFakeSSH_Outcomes(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.setState(func(s *fakeState) {
		s.SSH.Aliases = map[string]sshAlias{
			"github-work": {Login: "zamyb-work"},
			"github-me":   {Result: "denied"},
			"github-new":  {Result: "hostkey"},
			"github-slow": {Result: "timeout"},
		}
	})
	tests := []struct {
		alias  string
		code   int
		stderr string
	}{
		{"github-work", 1, "Hi zamyb-work! You've successfully authenticated, but GitHub does not provide shell access."},
		{"github-me", 255, "git@github.com: Permission denied (publickey)."},
		{"github-new", 255, "Host key verification failed."},
		{"github-slow", 255, "ssh: connect to host github.com port 22: Connection timed out"},
		{"github-none", 255, "ssh: Could not resolve hostname github-none: Name or service not known"},
	}
	for _, tt := range tests {
		res := sb.fake(sb.repoDir, "ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "git@"+tt.alias)
		assertCode(t, res, tt.code)
		if strings.TrimSpace(res.Stderr) != tt.stderr {
			t.Fatalf("%s stderr = %q, want %q", tt.alias, res.Stderr, tt.stderr)
		}
	}
	// Anything else, including host-key options, is rejected loudly.
	res := sb.fake(sb.repoDir, "ssh", "-T", "-o", "StrictHostKeyChecking=no", "git@github-work")
	assertCode(t, res, 64)
}

func TestFakeKeygen_CreatesPair(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	key := filepath.Join(sb.root, "id_test")
	res := sb.fake(sb.repoDir, "ssh-keygen", "-t", "ed25519", "-C", "zamyb-work", "-f", key, "-N", "")
	assertCode(t, res, 0)
	priv, err := os.ReadFile(key)
	if err != nil || string(priv) != "FAKE PRIVATE KEY zamyb-work\n" {
		t.Fatalf("private = %q, %v", priv, err)
	}
	pub, err := os.ReadFile(key + ".pub")
	if err != nil || string(pub) != "ssh-ed25519 AAAAFAKE zamyb-work\n" {
		t.Fatalf("public = %q, %v", pub, err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(key)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("private mode = %v", info.Mode().Perm())
		}
	}
	// Existing path is never overwritten.
	assertCode(t, sb.fake(sb.repoDir, "ssh-keygen", "-t", "ed25519", "-C", "x", "-f", key, "-N", ""), 1)

	sb.setState(func(s *fakeState) { s.Keygen.Fail = true })
	assertCode(t, sb.fake(sb.repoDir, "ssh-keygen", "-t", "ed25519", "-C", "x", "-f", key+"2", "-N", ""), 1)
}

func TestFakes_UnsupportedArgsExit64(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	for _, tc := range []struct {
		tool string
		args []string
	}{
		{"gh", []string{"repo", "list"}},
		{"git", []string{"push"}},
		{"git", []string{"config", "--list"}},
		{"ssh", []string{"git@github.com"}},
		{"ssh-keygen", []string{"-y", "-f", "x"}},
	} {
		res := sb.fake(sb.repoDir, tc.tool, tc.args...)
		assertCode(t, res, 64)
		assertContains(t, res.Stderr, "fake "+tc.tool+": unsupported arguments: "+strings.Join(tc.args, " "))
	}
	// Forced failures match substrings of the argument vector.
	sb.setState(func(s *fakeState) { s.Git.Fail = []string{"remote set-url"} })
	res := sb.fake(sb.repoDir, "git", "remote", "set-url", "origin", "x")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "fake git: forced failure (remote set-url)")
}
