package e2e

import "testing"

func TestImportAll_RejectsNonGitHubHostname(t *testing.T) {
	t.Parallel()
	for _, host := range []string{"ghe.example.com", "gitlab.com", "github.com.evil.example"} {
		sb := newSandbox(t)
		sb.seedAccounts("zamyb", "zamyb")
		res := sb.run("import-all", "--hostname", host)
		assertCode(t, res, 2)
		assertContains(t, res.Stderr, "only github.com is supported")
		if n := len(sb.calls()); n != 0 {
			t.Fatalf("%s: %d tool calls:\n%s", host, n, sb.logDump())
		}
		if sb.configExists() {
			t.Fatalf("%s: config written", host)
		}
	}
}

func TestImportAll_AcceptsGitHubCaseInsensitive(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedAccounts("zamyb", "zamyb", "zamyb-work")
	sb.setState(func(s *fakeState) {
		s.GH.Users = map[string]ghUser{
			"zamyb":      {ID: 2, Login: "zamyb", Name: "Izzam", Email: "me@example.com"},
			"zamyb-work": {ID: 1, Login: "zamyb-work", Name: "Izzam", Email: "work@example.com"},
		}
	})

	res := sb.run("import-all", "--hostname", "GitHub.com")
	assertCode(t, res, 0)
	if res.Stdout != "imported 2 profiles from gh host \"github.com\"\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	sb.assertSequence(
		mk("gh", "auth", "status", "--hostname", "github.com", "--json", "hosts"),
		switchTo("zamyb-work"),
		mk("gh", "api", "user"),
		switchTo("zamyb"),
	)
	if got := activeLogin(sb); got != "zamyb" {
		t.Fatalf("active = %s, want zamyb restored", got)
	}
	want := "[zamyb]\ngh_user = \"zamyb\"\ngit_name = \"Izzam\"\ngit_email = \"me@example.com\"\nssh_host_alias = \"github-zamyb\"\nssh_key = \"~/.ssh/id_ed25519_zamyb\"\n\n" +
		"[zamyb-work]\ngh_user = \"zamyb-work\"\ngit_name = \"Izzam\"\ngit_email = \"work@example.com\"\nssh_host_alias = \"github-zamyb-work\"\nssh_key = \"~/.ssh/id_ed25519_zamyb-work\"\n"
	if got := sb.readConfig(); got != want {
		t.Fatalf("config:\n%s\nwant:\n%s", got, want)
	}
}
