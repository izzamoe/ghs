package e2e

import (
	"strings"
	"testing"
)

const header = "   PROFILE  GH USER     GIT EMAIL         SSH ALIAS    WORKSPACE         GH AUTH\n"

func listSandbox(t *testing.T) *sandbox {
	sb := newSandbox(t)
	sb.writeConfig(cfgWork + "workspace = \"~/Documents/work\"\n\n" + cfgMe + "\n" + cfgOld)
	return sb
}

func TestList_MarksActiveAndAuthColumns(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")

	res := sb.run("list")
	assertCode(t, res, 0)
	want := header +
		"*  work     zamyb-work  work@example.com  github-work  ~/Documents/work  active\n" +
		"   me       zamyb       me@example.com    github-me    -                 yes\n" +
		"   old      olduser     (missing)         github-old   -                 no\n" +
		"active gh account: zamyb-work (profile \"work\")\n"
	if res.Stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", res.Stdout, want)
	}
}

func TestList_ActiveMatchesNoProfile(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.seedAccounts("stranger", "stranger", "zamyb")
	res := sb.run("list")
	assertCode(t, res, 0)
	assertNotContains(t, res.Stdout, "*")
	assertContains(t, res.Stdout, "   me       zamyb       me@example.com    github-me    -                 yes\n")
	if !strings.HasSuffix(res.Stdout, "active gh account: stranger (no profile matches)\n") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestList_WithoutGHShowsUnknown(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.without("gh")
	res := sb.run("list")
	assertCode(t, res, 0)
	for _, row := range strings.Split(strings.TrimSuffix(res.Stdout, "\n"), "\n")[1:4] {
		if !strings.HasSuffix(row, "?") {
			t.Fatalf("row %q does not end with ?", row)
		}
	}
	if !strings.HasSuffix(res.Stdout, "active gh account: unknown (gh not found)\n") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestList_GHFailureShowsUnknown(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.seedAccounts("zamyb-work", "zamyb-work")
	sb.setState(func(s *fakeState) { s.GH.Fail = []string{"auth status"} })
	res := sb.run("list")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "active gh account: unknown (fake gh: forced failure (auth status))\n")
	assertContains(t, res.Stdout, "~/Documents/work  ?\n")
}

func TestList_OtherHostsOnly(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = []ghAccount{{Login: "zamyb-work", Active: true, State: "success", Host: "ghe.example.com"}}
	})
	res := sb.run("list")
	assertCode(t, res, 0)
	assertNotContains(t, res.Stdout, "*")
	assertNotContains(t, res.Stdout, "active\n")
	assertContains(t, res.Stdout, "~/Documents/work  no\n")
	if !strings.HasSuffix(res.Stdout, "active gh account: none for github.com\n") {
		t.Fatalf("stdout = %q", res.Stdout)
	}
}

func TestList_ShowsWorkspaceColumn(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.seedAccounts("zamyb", "zamyb")
	res := sb.run("list")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "WORKSPACE")
	assertContains(t, res.Stdout, "github-work  ~/Documents/work  no\n")
	assertContains(t, res.Stdout, "github-me    -                 active\n")
}

func TestList_NoProfiles(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.seedAccounts("zamyb", "zamyb")
	res := sb.run("list")
	assertCode(t, res, 0)
	if res.Stdout != "no profiles found\nactive gh account: zamyb (no profile matches)\n" {
		t.Fatalf("stdout = %q", res.Stdout)
	}
	if sb.configExists() {
		t.Fatal("list created a config file")
	}
}

func TestList_NoMutations(t *testing.T) {
	t.Parallel()
	sb := listSandbox(t)
	sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")
	before := sb.snapshot(sb.root)
	delete(before, "calls.log")
	assertCode(t, sb.run("list"), 0)
	sb.assertNoMutations()
	after := sb.snapshot(sb.root)
	delete(after, "calls.log")
	assertSnapshotEqual(t, before, after)
}
