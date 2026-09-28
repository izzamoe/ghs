package e2e

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const wsKey = "includeIf.gitdir/i:~/Documents/work/.path"

const identityWork = "[user]\n\tname = Izzam\n\temail = work@example.com\n"

// workspaceSandbox has the standard profiles and an existing
// ~/Documents/work directory.
func workspaceSandbox(t *testing.T) *sandbox {
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")
	if err := os.MkdirAll(sb.expand("~/Documents/work"), 0o700); err != nil {
		t.Fatal(err)
	}
	return sb
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func globalValues(sb *sandbox, key string) []string {
	return sb.state().Git.Global[key]
}

func TestWorkspace_LinkWritesFileAddsIncludeSavesKey(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	file := sb.identityFile("work")

	res := sb.run("workspace", "work", "~/Documents/work")
	assertCode(t, res, 0)
	want := "wrote identity file " + file + "\n" +
		"added git include: " + wsKey + " = " + file + "\n" +
		"saved workspace ~/Documents/work for profile \"work\"\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}
	if got := readFile(t, filepath.FromSlash(file)); got != identityWork {
		t.Fatalf("identity file = %q", got)
	}
	sb.assertSequence(
		mk("git", "config", "--global", "--get-all", wsKey),
		mk("git", "config", "--global", "--add", wsKey, file),
	)
	if n := sb.countCalls("git", "--add"); n != 1 {
		t.Fatalf("%d --add calls, want exactly 1", n)
	}
	assertContains(t, sb.readConfig(), "[work]\ngh_user = \"zamyb-work\"\ngit_name = \"Izzam\"\ngit_email = \"work@example.com\"\nssh_host_alias = \"github-work\"\nssh_key = \"~/.ssh/id_ed25519_work\"\nworkspace = \"~/Documents/work\"\n")
	if got := globalValues(sb, wsKey); !slices.Equal(got, []string{file}) {
		t.Fatalf("global %s = %v", wsKey, got)
	}
	// ghs never edits ~/.gitconfig itself (FR-055).
	if _, err := os.Stat(sb.expand("~/.gitconfig")); !os.IsNotExist(err) {
		t.Fatalf("~/.gitconfig was created: %v", err)
	}
	for _, c := range sb.callsOf("git") {
		if !strings.HasPrefix(strings.Join(c.Args, " "), "config --global") {
			t.Fatalf("non-global git call %s", c)
		}
	}
}

func TestWorkspace_SecondRunIsIdempotent(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	file := filepath.FromSlash(sb.identityFile("work"))
	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(file, old, old); err != nil {
		t.Fatal(err)
	}
	config := sb.readConfig()

	res := sb.run("workspace", "work", "~/Documents/work/")
	assertCode(t, res, 0)
	if res.Stdout != "workspace ~/Documents/work is already linked for profile \"work\"\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if n := sb.countCalls("git", "--add"); n != 1 {
		t.Fatalf("%d --add calls over two runs, want 1", n)
	}
	if info, _ := os.Stat(file); !info.ModTime().Equal(old) {
		t.Fatal("up-to-date identity file was rewritten")
	}
	if sb.readConfig() != config {
		t.Fatal("config rewritten")
	}
}

func TestWorkspace_RequiresEmail(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	res := sb.run("workspace", "old", "~/Documents/old")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `profile "old" has no email; run: ghs set-email old <email>`)
	sb.assertNoMutations()
	if _, err := os.Stat(filepath.FromSlash(sb.identityFile("old"))); !os.IsNotExist(err) {
		t.Fatal("identity file written")
	}
}

func TestWorkspace_RejectsInvalidPath(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"workspace", "work", "relative/dir"},
		{"workspace", "work", "~"},
		{"workspace", "work", "~/"},
		{"workspace", "work", "/"},
		{"workspace", "work", "~/a\nb"},
		{"workspace", "work", `~/a"b`},
		{"workspace", "work", "~/a\x01b"},
		{"workspace", "work"},
		{"workspace", "work", "~/Documents/work", "--unlink"},
		{"add-profile", "work", "--gh-user", "zamyb-work", "--git-name", "Izzam", "--git-email", "work@example.com", "--ssh-alias", "github-work", "--ssh-key", "~/.ssh/id", "--workspace", "relative"},
	} {
		sb := workspaceSandbox(t)
		before := sb.readConfig()
		res := sb.run(args...)
		assertCode(t, res, 2)
		if len(sb.calls()) != 0 || sb.readConfig() != before {
			t.Fatalf("%q: side effects", args)
		}
	}
}

func TestWorkspace_AddProfileFlagLinksAfterSave(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	if err := os.MkdirAll(sb.expand("~/Documents/work"), 0o700); err != nil {
		t.Fatal(err)
	}
	file := sb.identityFile("work")

	res := sb.run(addWorkArgs("--workspace", "~/Documents/work")...)
	assertCode(t, res, 0)
	want := "saved profile \"work\" to " + sb.cfgPath() + "\n" +
		"wrote identity file " + file + "\n" +
		"added git include: " + wsKey + " = " + file + "\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}
	assertContains(t, sb.readConfig(), `workspace = "~/Documents/work"`)
	if got := readFile(t, filepath.FromSlash(file)); got != identityWork {
		t.Fatalf("identity file = %q", got)
	}
}

func TestWorkspace_AddFromGHFlagLinks(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedAccounts("zamyb-work", "zamyb-work")
	sb.setState(func(s *fakeState) {
		s.GH.Users = map[string]ghUser{"zamyb-work": {ID: 1, Login: "zamyb-work", Name: "Izzam", Email: "work@example.com"}}
	})

	res := sb.run("add-from-gh", "work", "--workspace", "~/Documents/work")
	assertCode(t, res, 0)
	file := sb.identityFile("work")
	assertContains(t, res.Stdout, "added git include: "+wsKey+" = "+file)
	assertContains(t, sb.readConfig(), `workspace = "~/Documents/work"`)
	if got := globalValues(sb, wsKey); !slices.Equal(got, []string{file}) {
		t.Fatalf("global %s = %v", wsKey, got)
	}
}

func TestWorkspace_AddProfileLinkFailureReportsRerun(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.setState(func(s *fakeState) { s.Git.Fail = []string{"config --global --add"} })

	res := sb.run(addWorkArgs("--workspace", "~/Documents/work")...)
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, "workspace link failed: ")
	assertContains(t, res.Stderr, "; rerun: ghs workspace work ~/Documents/work")
	assertContains(t, res.Stdout, `saved profile "work"`)
	assertContains(t, sb.readConfig(), `workspace = "~/Documents/work"`)

	// The rerun completes the link.
	sb.setState(func(s *fakeState) { s.Git.Fail = nil })
	res = sb.run("workspace", "work", "~/Documents/work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "added git include: ")
}

func TestWorkspace_UnlinkRemovesEntryFileAndKey(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	file := sb.identityFile("work")

	res := sb.run("workspace", "work", "--unlink")
	assertCode(t, res, 0)
	want := "removed git include: " + wsKey + " = " + file + "\n" +
		"deleted identity file " + file + "\n" +
		"removed workspace from profile \"work\"\n"
	if res.Stdout != want {
		t.Fatalf("stdout = %q, want %q", res.Stdout, want)
	}
	if n := sb.countCalls("git", "--unset --fixed-value"); n != 1 {
		t.Fatalf("%d --unset calls, want 1", n)
	}
	sb.assertSequence(mk("git", "config", "--global", "--unset", "--fixed-value", wsKey, file))
	if len(globalValues(sb, wsKey)) != 0 {
		t.Fatalf("include entry left: %v", globalValues(sb, wsKey))
	}
	if _, err := os.Stat(filepath.FromSlash(file)); !os.IsNotExist(err) {
		t.Fatal("identity file not deleted")
	}
	if strings.Contains(sb.readConfig(), "workspace") {
		t.Fatalf("workspace key left:\n%s", sb.readConfig())
	}
	if sb.countCalls("gh", "") != 0 || sb.sshConfig() != "" {
		t.Fatal("unlink touched gh or ssh")
	}
}

func TestWorkspace_UnlinkWhenAlreadyAbsentSucceeds(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork + "workspace = \"~/Documents/work\"\n")
	res := sb.run("workspace", "work", "--unlink")
	assertCode(t, res, 0)
	if res.Stdout != "removed workspace from profile \"work\"\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if strings.Contains(sb.readConfig(), "workspace") {
		t.Fatal("workspace key left")
	}

	res = sb.run("workspace", "work", "--unlink")
	assertCode(t, res, 0)
	if res.Stdout != "profile \"work\" has no workspace\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestWorkspace_LegacyKeyLoadsWithoutWarningAndListShowsIt(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork + "workspace = \"~/Documents/work\"\n")
	sb.without("gh")
	res := sb.run("list")
	assertCode(t, res, 0)
	if res.Stderr != "" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	assertContains(t, res.Stdout, "~/Documents/work")
}

func TestWorkspace_MissingDirectoryPrintsNote(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	res := sb.run("workspace", "work", "~/Documents/work")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "note: "+sb.expand("~/Documents/work")+" does not exist yet; the identity applies once repositories exist under it\n")
}

func TestWorkspace_SetEmailRefreshesIdentityFile(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	file := sb.identityFile("work")

	res := sb.run("set-email", "work", "new@example.com")
	assertCode(t, res, 0)
	if res.Stdout != "set email for profile \"work\"\nupdated identity file "+file+"\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if got := readFile(t, filepath.FromSlash(file)); got != "[user]\n\tname = Izzam\n\temail = new@example.com\n" {
		t.Fatalf("identity file = %q", got)
	}
	assertContains(t, sb.readConfig(), `workspace = "~/Documents/work"`)
}

func TestWorkspace_ImportAllOverwritePreservesWorkspaceAndRefreshesFile(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig("[zamyb-work]\ngh_user = \"zamyb-work\"\ngit_name = \"Izzam\"\ngit_email = \"old@example.com\"\nssh_host_alias = \"github-zamyb-work\"\nssh_key = \"~/.ssh/id_ed25519_zamyb-work\"\n")
	sb.seedAccounts("zamyb-work", "zamyb-work")
	assertCode(t, sb.run("workspace", "zamyb-work", "~/Documents/work"), 0)
	sb.setState(func(s *fakeState) {
		s.GH.Users = map[string]ghUser{"zamyb-work": {ID: 1, Login: "zamyb-work", Name: "Izzam Z", Email: "work@example.com"}}
	})

	res := sb.run("import-all")
	assertCode(t, res, 0)
	assertContains(t, sb.readConfig(), `workspace = "~/Documents/work"`)
	assertContains(t, sb.readConfig(), `git_email = "work@example.com"`)
	if got := readFile(t, filepath.FromSlash(sb.identityFile("zamyb-work"))); got != "[user]\n\tname = Izzam Z\n\temail = work@example.com\n" {
		t.Fatalf("identity file = %q", got)
	}
	assertContains(t, res.Stdout, "updated identity file "+sb.identityFile("zamyb-work"))
}

func TestWorkspace_ForeignIncludeEntryUntouched(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	sb.setState(func(s *fakeState) {
		s.Git.Global = map[string]gitValues{wsKey: {"/somewhere/else/gitconfig"}}
	})
	file := sb.identityFile("work")

	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	if got := globalValues(sb, wsKey); !slices.Equal(got, []string{"/somewhere/else/gitconfig", file}) {
		t.Fatalf("after link: %v", got)
	}
	assertCode(t, sb.run("workspace", "work", "--unlink"), 0)
	if got := globalValues(sb, wsKey); !slices.Equal(got, []string{"/somewhere/else/gitconfig"}) {
		t.Fatalf("after unlink: %v", got)
	}
}

func TestWorkspace_RejectsSameAndNestedDirectory(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	before := sb.readConfig()
	for _, path := range []string{"~/Documents/work", "~/Documents/work/sub", "~/Documents", "~/documents/WORK"} {
		logBefore := len(sb.calls())
		res := sb.run("workspace", "me", path)
		assertCode(t, res, 1)
		assertContains(t, res.Stderr, `overlaps profile "work" (~/Documents/work)`)
		if sb.readConfig() != before {
			t.Fatalf("%s: config changed", path)
		}
		for _, c := range sb.calls()[logBefore:] {
			if isMutating(c) {
				t.Fatalf("%s: mutating call %s", path, c)
			}
		}
	}
	if _, err := os.Stat(filepath.FromSlash(sb.identityFile("me"))); !os.IsNotExist(err) {
		t.Fatal("identity file for me written")
	}
}

func TestWorkspace_RepairsPartialLink(t *testing.T) {
	t.Parallel()
	t.Run("file only", func(t *testing.T) {
		t.Parallel()
		sb := workspaceSandbox(t)
		assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
		sb.setState(func(s *fakeState) { delete(s.Git.Global, wsKey) })

		res := sb.run("workspace", "work", "~/Documents/work")
		assertCode(t, res, 0)
		if res.Stdout != "added git include: "+wsKey+" = "+sb.identityFile("work")+"\n" {
			t.Fatalf("stdout = %q", res.Stdout)
		}
	})
	t.Run("entry only", func(t *testing.T) {
		t.Parallel()
		sb := workspaceSandbox(t)
		assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
		if err := os.Remove(filepath.FromSlash(sb.identityFile("work"))); err != nil {
			t.Fatal(err)
		}
		res := sb.run("workspace", "work", "~/Documents/work")
		assertCode(t, res, 0)
		if res.Stdout != "wrote identity file "+sb.identityFile("work")+"\n" {
			t.Fatalf("stdout = %q", res.Stdout)
		}
		if n := sb.countCalls("git", "--add"); n != 1 {
			t.Fatalf("%d --add calls, want 1", n)
		}
	})
}

func TestWorkspace_WindowsBackslashPathNormalized(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgWork)
	res := sb.run("workspace", "work", `C:\Users\x\work`)
	assertCode(t, res, 0)
	key := "includeIf.gitdir/i:C:/Users/x/work/.path"
	assertContains(t, res.Stdout, "added git include: "+key+" = "+sb.identityFile("work"))
	assertContains(t, sb.readConfig(), `workspace = "C:/Users/x/work"`)
	if got := globalValues(sb, key); len(got) != 1 {
		t.Fatalf("global %s = %v", key, got)
	}
}

func TestWorkspace_IdentityAppliesInsideWorkspace(t *testing.T) {
	t.Parallel()
	sb := workspaceSandbox(t)
	assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
	repo := sb.expand("~/Documents/work/app")
	if err := os.MkdirAll(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	sb.setState(func(s *fakeState) { s.Git.Toplevel = repo })
	res := sb.fake(repo, "git", "config", "--show-origin", "--show-scope", "--get", "user.email")
	if res.Stdout != "global\tfile:"+sb.identityFile("work")+"\twork@example.com\n" {
		t.Fatalf("git resolves %q inside the workspace", res.Stdout)
	}
}
