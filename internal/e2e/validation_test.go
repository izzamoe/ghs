package e2e

import (
	"strings"
	"testing"
)

// withFlag returns the standard add-profile arguments with one flag's value
// replaced (or the name replaced when flag is "name").
func withFlag(flag, value string) []string {
	args := addWorkArgs()
	if flag == "name" {
		args[1] = value
		return args
	}
	for i := range args {
		if args[i] == "--"+flag {
			// Use --flag=value so values starting with "-" reach validation.
			args = append(args[:i], args[i+2:]...)
			return append(args, "--"+flag+"="+value)
		}
	}
	panic("unknown flag " + flag)
}

func TestValidation_AddProfileRejectsMalformed(t *testing.T) {
	t.Parallel()
	cases := []struct{ flag, value string }{
		{"name", "wo]rk"},
		{"name", "[work"},
		{"name", "my work"},
		{"name", "a/b"},
		{"name", "a\x01b"},
		{"name", "-work"},
		{"name", ".work"},
		{"name", "help"},
		{"name", "list"},
		{"name", strings.Repeat("n", 65)},
		{"ssh-alias", "github work"},
		{"ssh-alias", "a#b"},
		{"ssh-alias", `a"b`},
		{"ssh-alias", "github.com"},
		{"ssh-alias", "-alias"},
		{"gh-user", "-bad-"},
		{"gh-user", strings.Repeat("a", 40)},
		{"gh-user", "bad\nname"},
		{"git-name", `Izzam "Z"`},
		{"git-name", "Izzam\rX"},
		{"git-email", "a@b"},
		{"git-email", "@x.com"},
		{"git-email", "a b@x.com"},
		{"git-email", "a@x.com\n"},
		{"ssh-key", "relative/key"},
		{"ssh-key", `~/a"b`},
		{"ssh-key", "~/key\nIdentityFile /etc/passwd"},
	}
	for _, tc := range cases {
		t.Run(tc.flag+"="+tc.value, func(t *testing.T) {
			t.Parallel()
			sb := newSandbox(t)
			res := sb.run(withFlag(tc.flag, tc.value)...)
			assertCode(t, res, 2)
			if !strings.HasPrefix(res.Stderr, "ghs: ") {
				t.Fatalf("stderr = %q, want ghs: prefix", res.Stderr)
			}
			if sb.configExists() {
				t.Fatalf("config was written:\n%s", sb.readConfig())
			}
			if n := len(sb.calls()); n != 0 {
				t.Fatalf("%d tool calls on a usage error:\n%s", n, sb.logDump())
			}
		})
	}
	// An empty value is a usage error from the parser as well.
	sb := newSandbox(t)
	assertCode(t, sb.run(append(addWorkArgs()[:10], "--ssh-key=")...), 2)
}

func TestValidation_GHValuesRejected(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		user  ghUser
		field string
	}{
		{"login with newline", ghUser{ID: 1, Login: "bad\nname", Name: "Izzam", Email: "a@x.com"}, "login"},
		{"name with quote", ghUser{ID: 1, Login: "zamyb", Name: `Izzam "Z"`, Email: "a@x.com"}, "git name"},
		{"malformed email", ghUser{ID: 1, Login: "zamyb", Name: "Izzam", Email: "not-an-email"}, "email"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sb := newSandbox(t)
			sb.seedAccounts("zamyb", "zamyb")
			sb.setState(func(s *fakeState) { s.GH.Users = map[string]ghUser{"zamyb": tc.user} })

			res := sb.run("add-from-gh", "me")
			assertCode(t, res, 1)
			assertContains(t, res.Stderr, tc.field)
			if sb.configExists() {
				t.Fatalf("config was written:\n%s", sb.readConfig())
			}
		})
	}
}

func TestValidation_CaseInsensitiveDuplicateNameSuggests(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)

	res := sb.run("add-profile", "Work", "--gh-user", "zamyb", "--git-name", "Izzam", "--git-email", "me@example.com", "--ssh-alias", "github-other", "--ssh-key", "~/.ssh/id_other")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `"work"`)
	if got := sb.readConfig(); got != cfgWork {
		t.Fatalf("config changed:\n%s", got)
	}
}

func TestValidation_AliasCollisionRejected(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)

	res := sb.run("add-profile", "me", "--gh-user", "zamyb", "--git-name", "Izzam", "--git-email", "me@example.com", "--ssh-alias", "GitHub-Work", "--ssh-key", "~/.ssh/id_me")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `profile "work"`)
	if got := sb.readConfig(); got != cfgWork {
		t.Fatalf("config changed:\n%s", got)
	}

	// Re-adding the same profile name keeps working (it replaces itself).
	assertCode(t, sb.run(addWorkArgs()...), 0)
}

func TestValidation_DerivedAliasCollision(t *testing.T) {
	t.Parallel()
	const taken = "[other]\ngh_user = \"otheruser\"\ngit_name = \"O\"\nssh_host_alias = \"github-zamyb\"\nssh_key = \"~/.ssh/id_other\"\n"

	t.Run("add-from-gh", func(t *testing.T) {
		t.Parallel()
		sb := newSandbox(t)
		sb.writeConfig(taken)
		sb.seedAccounts("zamyb", "zamyb")
		sb.setState(func(s *fakeState) {
			s.GH.Users = map[string]ghUser{"zamyb": {ID: 2, Login: "zamyb", Name: "Izzam", Email: "me@example.com"}}
		})
		res := sb.run("add-from-gh", "zamyb")
		assertCode(t, res, 1)
		assertContains(t, res.Stderr, "github-zamyb")
		assertContains(t, res.Stderr, "--ssh-alias")
		if got := sb.readConfig(); got != taken {
			t.Fatalf("config changed:\n%s", got)
		}
	})

	t.Run("import-all", func(t *testing.T) {
		t.Parallel()
		sb := newSandbox(t)
		sb.writeConfig(taken)
		sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")
		sb.setState(func(s *fakeState) {
			s.GH.Users = map[string]ghUser{
				"zamyb-work": {ID: 1, Login: "zamyb-work", Name: "Izzam", Email: "work@example.com"},
				"zamyb":      {ID: 2, Login: "zamyb", Name: "Izzam", Email: "me@example.com"},
			}
		})
		res := sb.run("import-all")
		assertCode(t, res, 1)
		assertContains(t, res.Stderr, "github-zamyb")
		if got := sb.readConfig(); got != taken {
			t.Fatalf("config changed:\n%s", got)
		}
		gh := sb.callsOf("gh")
		last := gh[len(gh)-1]
		if last.String() != "gh auth switch --hostname github.com --user zamyb-work" {
			t.Fatalf("last gh call = %s, want the restore switch\n%s", last, sb.logDump())
		}
		for _, a := range sb.state().GH.Accounts {
			if a.Active != (a.Login == "zamyb-work") {
				t.Fatalf("active account not restored: %+v", sb.state().GH.Accounts)
			}
		}
	})
}

func TestValidation_SetEmailRejectsMalformed(t *testing.T) {
	t.Parallel()
	for _, email := range []string{"a@b", "@x.com", "a b@x.com", "a@x.com\n", `a"b@x.com`, "a@@x.com"} {
		t.Run(email, func(t *testing.T) {
			t.Parallel()
			sb := newSandbox(t)
			sb.writeConfig(cfgWork)
			res := sb.run("set-email", "work", email)
			assertCode(t, res, 2)
			if got := sb.readConfig(); got != cfgWork {
				t.Fatalf("config changed:\n%s", got)
			}
			if len(sb.calls()) != 0 {
				t.Fatalf("tools invoked:\n%s", sb.logDump())
			}
		})
	}
}
