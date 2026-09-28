package e2e

import (
	"os"
	"runtime"
	"testing"
)

const invalidLine3 = "[work]\ngh_user = \"zamyb-work\"\nthis line is not valid\n"

func TestConfig_InvalidLineStopsEveryWriter(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		args []string
	}{
		{"add-profile", []string{"add-profile", "new", "--gh-user", "newuser", "--git-name", "New", "--git-email", "new@example.com", "--ssh-alias", "github-new", "--ssh-key", "~/.ssh/id_new"}},
		{"add-from-gh", []string{"add-from-gh", "new"}},
		{"import-all", []string{"import-all"}},
		{"set-email", []string{"set-email", "work", "x@example.com"}},
		{"workspace", []string{"workspace", "work", "~/Documents/work"}},
		{"workspace --unlink", []string{"workspace", "work", "--unlink"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sb := newSandbox(t)
			sb.writeConfig(invalidLine3)
			sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")
			sb.setState(func(s *fakeState) {
				s.GH.Users = map[string]ghUser{"zamyb-work": {ID: 101, Login: "zamyb-work", Name: "Izzam", Email: "work@example.com"}}
			})
			ghBefore := sb.state().GH

			res := sb.run(tc.args...)
			assertCode(t, res, 1)
			assertContains(t, res.Stderr, "line 3")
			if got := sb.readConfig(); got != invalidLine3 {
				t.Fatalf("config changed:\n%s", got)
			}
			sb.assertNoMutations()
			if after := sb.state().GH; len(after.Accounts) != len(ghBefore.Accounts) || !after.Accounts[0].Active {
				t.Fatalf("active account changed: %+v", after.Accounts)
			}
		})
	}
}

func TestConfig_UnreadableFileNotReplaced(t *testing.T) {
	t.Parallel()
	skipUnlessPermissionsEnforced(t)
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	if err := os.Chmod(sb.cfgPath(), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(sb.cfgPath(), 0o600) })

	res := sb.run("add-profile", "me", "--gh-user", "zamyb", "--git-name", "Izzam", "--git-email", "me@example.com", "--ssh-alias", "github-me", "--ssh-key", "~/.ssh/id_me")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "cannot read config")

	info, err := os.Stat(sb.cfgPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o000 {
		t.Fatalf("mode = %v, want 0000 (file must not be replaced)", info.Mode().Perm())
	}
	_ = os.Chmod(sb.cfgPath(), 0o600)
	if got := sb.readConfig(); got != cfgWork {
		t.Fatalf("config changed:\n%s", got)
	}
}

func TestConfig_AppendPreservesOrderAndFields(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	existing := cfgMe + "future_key = \"kept\"\n\n" + cfgOld
	sb.writeConfig(existing)

	res := sb.run(addWorkArgs()...)
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, `saved profile "work" to `+sb.cfgPath())
	if got, want := sb.readConfig(), existing+"\n"+cfgWork; got != want {
		t.Fatalf("config:\n%s\nwant:\n%s", got, want)
	}
}

func TestConfig_CreatesDirAndFileOwnerOnly(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	if _, err := os.Stat(sb.cfgDir()); !os.IsNotExist(err) {
		t.Fatalf("config dir already exists: %v", err)
	}

	res := sb.run(addWorkArgs()...)
	assertCode(t, res, 0)
	if got := sb.readConfig(); got != cfgWork {
		t.Fatalf("config:\n%s", got)
	}
	if runtime.GOOS == "windows" {
		return
	}
	for path, want := range map[string]os.FileMode{sb.cfgDir(): 0o700, sb.cfgPath(): 0o600} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != want {
			t.Fatalf("%s mode = %v, want %v", path, info.Mode().Perm(), want)
		}
	}
}

func TestConfig_UnknownKeysSurviveRoundTrip(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig("[work]\nfuture_b = \"2\"\ngh_user = \"zamyb-work\"\ngit_name = \"Izzam\"\ngit_email = \"work@example.com\"\nssh_host_alias = \"github-work\"\nssh_key = \"~/.ssh/id_ed25519_work\"\nfuture_a = 1\n")

	res := sb.run("set-email", "work", "new@example.com")
	assertCode(t, res, 0)
	want := "[work]\ngh_user = \"zamyb-work\"\ngit_name = \"Izzam\"\ngit_email = \"new@example.com\"\nssh_host_alias = \"github-work\"\nssh_key = \"~/.ssh/id_ed25519_work\"\nfuture_b = \"2\"\nfuture_a = \"1\"\n"
	if got := sb.readConfig(); got != want {
		t.Fatalf("config:\n%s\nwant:\n%s", got, want)
	}
}
