package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// doctorSandbox is a machine where everything matches profile "work".
func doctorSandbox(t *testing.T) *sandbox {
	sb := newSandbox(t)
	sb.seedStandard()
	sb.setState(func(s *fakeState) {
		s.Git.Origin = "git@github-work:acme/app.git"
		s.Git.Local = map[string]gitValues{"user.name": {"Izzam"}, "user.email": {"work@example.com"}}
		s.SSH.Aliases = map[string]sshAlias{"github-work": {Login: "zamyb-work"}, "github-me": {Login: "zamyb"}}
	})
	sb.touchKey("~/.ssh/id_ed25519_work")
	sb.touchKey("~/.ssh/id_ed25519_me")
	sb.writeSSHConfig(sshBlock("github-work", "~/.ssh/id_ed25519_work") + "\n" + sshBlock("github-me", "~/.ssh/id_ed25519_me"))
	return sb
}

func (sb *sandbox) localConfigFile() string {
	return filepath.ToSlash(sb.repoDir) + "/.git/config"
}

// doctorLine returns the output line for a check, e.g. "ssh auth".
func doctorLine(t *testing.T, stdout, check string) string {
	t.Helper()
	for line := range strings.SplitSeq(stdout, "\n") {
		if len(line) > 6 && strings.HasPrefix(line[6:], check+":") {
			return line
		}
	}
	t.Fatalf("no %q line in:\n%s", check, stdout)
	return ""
}

func TestDoctor_AllOKExit0(t *testing.T) {
	t.Parallel()
	sb := doctorSandbox(t)
	res := sb.run("doctor")
	assertCode(t, res, 0)
	want := "ghs doctor: profile \"work\"\n" +
		"ok    gh: active account zamyb-work (profile \"work\")\n" +
		"ok    git identity: Izzam <work@example.com> (local, " + sb.localConfigFile() + ")\n" +
		"ok    origin: git@github-work:acme/app.git uses alias github-work\n" +
		"ok    ssh config: Host github-work -> github.com, IdentityFile " + sb.expand("~/.ssh/id_ed25519_work") + "\n" +
		"ok    ssh auth: github authenticated github-work as zamyb-work\n" +
		"skip  workspace: profile has no workspace\n" +
		"summary: 5 ok, 0 warn, 0 fail, 1 skip\n"
	if res.Stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", res.Stdout, want)
	}
	if res.Stderr != "" {
		t.Fatalf("stderr = %q", res.Stderr)
	}
	sb.assertSequence(mk("ssh", "-T", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "git@github-work"))
}

func TestDoctor_DefaultProfileFromActiveAccount(t *testing.T) {
	t.Parallel()
	sb := doctorSandbox(t)
	sb.seedAccounts("zamyb", "zamyb", "zamyb-work")
	res := sb.run("doctor", "--offline")
	if !strings.HasPrefix(res.Stdout, "ghs doctor: profile \"me\"\nok    gh: active account zamyb (profile \"me\")\n") {
		t.Fatalf("stdout:\n%s", res.Stdout)
	}
}

func TestDoctor_NoMatchingProfileSkipsRest(t *testing.T) {
	t.Parallel()
	sb := doctorSandbox(t)
	sb.seedAccounts("stranger", "stranger")
	res := sb.run("doctor")
	assertCode(t, res, 1)
	want := "ghs doctor: no profile selected\n" +
		"fail  gh: active account stranger matches no profile; run: ghs use <profile>\n" +
		"skip  git identity: no profile selected\n" +
		"skip  origin: no profile selected\n" +
		"skip  ssh config: no profile selected\n" +
		"skip  ssh auth: no profile selected\n" +
		"skip  workspace: no profile selected\n" +
		"summary: 0 ok, 0 warn, 1 fail, 5 skip\n"
	if res.Stdout != want {
		t.Fatalf("stdout:\n%s\nwant:\n%s", res.Stdout, want)
	}
	assertContains(t, res.Stderr, "doctor found 1 failing check")

	sb.seedAccounts("", "zamyb")
	res = sb.run("doctor")
	assertCode(t, res, 1)
	assertContains(t, res.Stdout, "fail  gh: no active account for github.com; run: gh auth login\n")
}

func TestDoctor_ExplicitProfileGHMismatchNamesBoth(t *testing.T) {
	t.Parallel()
	sb := doctorSandbox(t)
	res := sb.run("doctor", "me")
	assertCode(t, res, 1)
	assertContains(t, res.Stdout, "ghs doctor: profile \"me\"\n")
	assertContains(t, res.Stdout, "fail  gh: active account zamyb-work is not profile \"me\" (zamyb); run: ghs use me\n")
	// Every other check is evaluated against "me".
	assertContains(t, res.Stdout, "fail  git identity: email work@example.com differs from profile me@example.com (local, "+sb.localConfigFile()+"); run: ghs use me\n")
	assertContains(t, res.Stdout, "fail  origin: git@github-work:acme/app.git uses alias of profile \"work\"; run: ghs fix-remote me\n")
	assertContains(t, res.Stdout, "ok    ssh auth: github authenticated github-me as zamyb\n")
}

func TestDoctor_IdentityEmailFailNameWarn(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		local map[string]gitValues
		want  string
	}{
		{"email differs", map[string]gitValues{"user.name": {"Izzam"}, "user.email": {"other@example.com"}},
			"fail  git identity: email other@example.com differs from profile work@example.com (local, FILE); run: ghs use work"},
		{"name differs", map[string]gitValues{"user.name": {"Other"}, "user.email": {"work@example.com"}},
			"warn  git identity: name \"Other\" differs from profile \"Izzam\" (local, FILE)"},
		{"email unset", map[string]gitValues{"user.name": {"Izzam"}},
			"fail  git identity: user.email is not set (local); run: ghs use work"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sb := doctorSandbox(t)
			sb.setState(func(s *fakeState) { s.Git.Local = tc.local })
			res := sb.run("doctor", "--offline")
			want := strings.ReplaceAll(tc.want, "FILE", sb.localConfigFile())
			if got := doctorLine(t, res.Stdout, "git identity"); got != want {
				t.Fatalf("line = %q, want %q", got, want)
			}
		})
	}
	// Outside a repository the global identity is checked.
	sb := doctorSandbox(t)
	sb.setState(func(s *fakeState) {
		s.Git.Repo = false
		s.Git.Global = map[string]gitValues{"user.name": {"Izzam"}, "user.email": {"me@example.com"}}
	})
	res := sb.run("doctor", "--offline")
	want := "fail  git identity: email me@example.com differs from profile work@example.com (global, " + filepath.ToSlash(sb.home) + "/.gitconfig); run: ghs use work"
	if got := doctorLine(t, res.Stdout, "git identity"); got != want {
		t.Fatalf("line = %q, want %q", got, want)
	}
}

func TestDoctor_OriginVariants(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		origin string
		repo   bool
		want   string
	}{
		{"git@github.com:acme/app.git", true, "warn  origin: git@github.com:acme/app.git uses github.com directly; run: ghs fix-remote work"},
		{"git@github-me:acme/app.git", true, "fail  origin: git@github-me:acme/app.git uses alias of profile \"me\"; run: ghs fix-remote work"},
		{"git@some-alias:acme/app.git", true, "fail  origin: git@some-alias:acme/app.git uses unknown alias some-alias; run: ghs fix-remote work"},
		{"https://gitlab.com/g/app.git", true, "skip  origin: https://gitlab.com/g/app.git is not a GitHub remote (host gitlab.com)"},
		{"git@github.com:acme/app.git", false, "skip  origin: not inside a git repository"},
		{"", true, "skip  origin: no origin remote"},
		{"git@github-work:acme/app.git", true, "ok    origin: git@github-work:acme/app.git uses alias github-work"},
	} {
		t.Run(tc.want[:4]+" "+tc.origin, func(t *testing.T) {
			t.Parallel()
			sb := doctorSandbox(t)
			sb.setState(func(s *fakeState) {
				s.Git.Origin = tc.origin
				s.Git.Repo = tc.repo
			})
			res := sb.run("doctor", "--offline")
			if got := doctorLine(t, res.Stdout, "origin"); got != tc.want {
				t.Fatalf("line = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDoctor_SSHConfigVariants(t *testing.T) {
	t.Parallel()
	key := func(sb *sandbox) string { return sb.expand("~/.ssh/id_ed25519_work") }
	for _, tc := range []struct {
		name   string
		config func(sb *sandbox) string // "" = no file
		setup  func(sb *sandbox)
		want   func(sb *sandbox) string
	}{
		{"no file", nil, nil, func(sb *sandbox) string {
			return "fail  ssh config: no Host block for github-work in " + sb.sshCfgPath() + "; run: ghs init-ssh work"
		}},
		{"no block", func(*sandbox) string { return sshBlock("github-me", "~/.ssh/id_ed25519_me") }, nil, func(sb *sandbox) string {
			return "fail  ssh config: no Host block for github-work in " + sb.sshCfgPath() + "; run: ghs init-ssh work"
		}},
		{"missing HostName", func(*sandbox) string {
			return "Host github-work\n  User git\n  IdentityFile ~/.ssh/id_ed25519_work\n  IdentitiesOnly yes\n"
		}, nil, func(*sandbox) string {
			return "fail  ssh config: Host github-work is missing HostName github.com; run: ghs init-ssh work"
		}},
		{"missing User", func(*sandbox) string {
			return "Host github-work\n  HostName github.com\n  IdentityFile ~/.ssh/id_ed25519_work\n  IdentitiesOnly yes\n"
		}, nil, func(*sandbox) string {
			return "fail  ssh config: Host github-work is missing User git; run: ghs init-ssh work"
		}},
		{"missing IdentitiesOnly", func(*sandbox) string {
			return "Host github-work\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_work\n"
		}, nil, func(*sandbox) string {
			return "fail  ssh config: Host github-work is missing IdentitiesOnly yes; run: ghs init-ssh work"
		}},
		{"missing IdentityFile", func(*sandbox) string {
			return "Host github-work\n  HostName github.com\n  User git\n  IdentitiesOnly yes\n"
		}, nil, func(*sandbox) string {
			return "fail  ssh config: Host github-work is missing IdentityFile ~/.ssh/id_ed25519_work; run: ghs init-ssh work"
		}},
		{"IdentityFile does not exist", func(*sandbox) string { return sshBlock("github-work", "~/.ssh/id_wrong") }, nil, func(sb *sandbox) string {
			return "fail  ssh config: IdentityFile " + sb.expand("~/.ssh/id_wrong") + " does not exist; run: ghs init-ssh work"
		}},
		{"no .pub", func(*sandbox) string { return sshBlock("github-work", "~/.ssh/id_ed25519_work") }, func(sb *sandbox) {
			_ = os.Remove(key(sb) + ".pub")
		}, func(sb *sandbox) string {
			return "fail  ssh config: IdentityFile " + key(sb) + " has no " + key(sb) + ".pub; run: ghs init-ssh work"
		}},
		{"ok with quoted path", func(*sandbox) string { return sshBlock("github-work", `"~/My Keys/id work"`) }, func(sb *sandbox) {
			sb.touchKey("~/My Keys/id work")
		}, func(sb *sandbox) string {
			return "ok    ssh config: Host github-work -> github.com, IdentityFile " + sb.expand("~/My Keys/id work")
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sb := doctorSandbox(t)
			_ = os.Remove(sb.sshCfgPath())
			if tc.config != nil {
				sb.writeSSHConfig(tc.config(sb))
			}
			if tc.setup != nil {
				tc.setup(sb)
			}
			res := sb.run("doctor", "--offline")
			assertCode(t, res, map[bool]int{true: 0, false: 1}[strings.HasPrefix(tc.want(sb), "ok")])
			if got := doctorLine(t, res.Stdout, "ssh config"); got != tc.want(sb) {
				t.Fatalf("line = %q, want %q", got, tc.want(sb))
			}
		})
	}
	t.Run("unreadable", func(t *testing.T) {
		t.Parallel()
		skipUnlessPermissionsEnforced(t)
		sb := doctorSandbox(t)
		if err := os.Chmod(sb.sshCfgPath(), 0o000); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(sb.sshCfgPath(), 0o600) })
		res := sb.run("doctor", "--offline")
		assertCode(t, res, 1)
		assertContains(t, doctorLine(t, res.Stdout, "ssh config"), "fail  ssh config: cannot read "+sb.sshCfgPath()+": ")
		// Other checks still ran.
		assertContains(t, res.Stdout, "ok    origin:")
	})
}

func TestDoctor_SSHAuthVariants(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		alias sshAlias
		known bool
		want  string
	}{
		{sshAlias{Login: "zamyb-work"}, true, "ok    ssh auth: github authenticated github-work as zamyb-work"},
		{sshAlias{Login: "zamyb"}, true, "fail  ssh auth: github authenticated github-work as zamyb, expected zamyb-work"},
		{sshAlias{Result: "denied"}, true, "fail  ssh auth: git@github.com: Permission denied (publickey).; run once: ssh -T git@github-work"},
		{sshAlias{Result: "hostkey"}, true, "fail  ssh auth: Host key verification failed.; run once: ssh -T git@github-work"},
		{sshAlias{Result: "timeout"}, true, "fail  ssh auth: ssh: connect to host github.com port 22: Connection timed out; run once: ssh -T git@github-work"},
		{sshAlias{}, false, "fail  ssh auth: ssh: Could not resolve hostname github-work: Name or service not known; run once: ssh -T git@github-work"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			t.Parallel()
			sb := doctorSandbox(t)
			sb.setState(func(s *fakeState) {
				if tc.known {
					s.SSH.Aliases["github-work"] = tc.alias
				} else {
					delete(s.SSH.Aliases, "github-work")
				}
			})
			res := sb.run("doctor")
			if got := doctorLine(t, res.Stdout, "ssh auth"); got != tc.want {
				t.Fatalf("line = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDoctor_OfflineSkipsSSHAuth(t *testing.T) {
	t.Parallel()
	sb := doctorSandbox(t)
	res := sb.run("doctor", "--offline")
	assertCode(t, res, 0)
	if got := doctorLine(t, res.Stdout, "ssh auth"); got != "skip  ssh auth: --offline" {
		t.Fatalf("line = %q", got)
	}
	if n := len(sb.callsOf("ssh")); n != 0 {
		t.Fatalf("%d ssh calls with --offline", n)
	}
	assertContains(t, res.Stdout, "summary: 4 ok, 0 warn, 0 fail, 2 skip\n")
}

func TestDoctor_WorkspaceVariants(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		state string
		setup func(sb *sandbox)
		want  func(sb *sandbox) string
	}{
		{"none", func(*sandbox) {}, func(*sandbox) string { return "skip  workspace: profile has no workspace" }},
		{"configured-only", func(sb *sandbox) {
			sb.writeConfig(cfgWork + "workspace = \"~/Documents/work\"\n\n" + cfgMe + "\n" + cfgOld)
		}, func(*sandbox) string {
			return "fail  workspace: ~/Documents/work is configured but not linked; run: ghs workspace work ~/Documents/work"
		}},
		{"file-only", func(sb *sandbox) {
			assertCode(sb.t, sb.run("workspace", "work", "~/Documents/work"), 0)
			sb.setState(func(s *fakeState) { delete(s.Git.Global, wsKey) })
		}, func(*sandbox) string {
			return "fail  workspace: ~/Documents/work is configured but not linked; run: ghs workspace work ~/Documents/work"
		}},
		{"entry-only", func(sb *sandbox) {
			assertCode(sb.t, sb.run("workspace", "work", "~/Documents/work"), 0)
			_ = os.Remove(filepath.FromSlash(sb.identityFile("work")))
		}, func(sb *sandbox) string {
			return "fail  workspace: identity file " + sb.identityFile("work") + " is missing; run: ghs workspace work ~/Documents/work"
		}},
		{"stale", func(sb *sandbox) {
			assertCode(sb.t, sb.run("workspace", "work", "~/Documents/work"), 0)
			_ = os.WriteFile(filepath.FromSlash(sb.identityFile("work")), []byte("[user]\n\tname = Izzam\n\temail = old@example.com\n"), 0o600)
		}, func(sb *sandbox) string {
			return "fail  workspace: identity file " + sb.identityFile("work") + " differs from profile; run: ghs workspace work ~/Documents/work"
		}},
		{"linked", func(sb *sandbox) {
			assertCode(sb.t, sb.run("workspace", "work", "~/Documents/work"), 0)
		}, func(sb *sandbox) string {
			return "ok    workspace: ~/Documents/work linked via " + sb.identityFile("work")
		}},
	} {
		t.Run(tc.state, func(t *testing.T) {
			t.Parallel()
			sb := doctorSandbox(t)
			tc.setup(sb)
			res := sb.run("doctor", "--offline")
			if got := doctorLine(t, res.Stdout, "workspace"); got != tc.want(sb) {
				t.Fatalf("line = %q, want %q", got, tc.want(sb))
			}
		})
	}
}

func TestDoctor_NoMutations(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"doctor"}, {"doctor", "me"}, {"doctor", "--offline"}} {
		sb := doctorSandbox(t)
		assertCode(t, sb.run("workspace", "work", "~/Documents/work"), 0)
		sb.setState(func(s *fakeState) { s.Git.Origin = "git@github.com:acme/app.git" })
		if err := os.Truncate(sb.logPath, 0); err != nil {
			t.Fatal(err)
		}
		before := sb.snapshot(sb.root)
		delete(before, "calls.log")

		sb.run(args...)
		sb.assertNoMutations()
		after := sb.snapshot(sb.root)
		delete(after, "calls.log")
		assertSnapshotEqual(t, before, after)
	}
}

func TestDoctor_MissingToolsFailButContinue(t *testing.T) {
	t.Parallel()
	t.Run("gh", func(t *testing.T) {
		t.Parallel()
		sb := doctorSandbox(t)
		sb.without("gh")
		res := sb.run("doctor", "work", "--offline")
		assertCode(t, res, 1)
		assertContains(t, res.Stdout, "fail  gh: gh is not installed; see https://cli.github.com\n")
		assertContains(t, res.Stdout, "ok    git identity:")
		assertContains(t, res.Stdout, "ok    ssh config:")
		res = sb.run("doctor", "--offline")
		assertContains(t, res.Stdout, "skip  git identity: no profile selected\n")
	})
	t.Run("git", func(t *testing.T) {
		t.Parallel()
		sb := doctorSandbox(t)
		sb.without("git")
		res := sb.run("doctor", "--offline")
		assertCode(t, res, 1)
		assertContains(t, res.Stdout, "fail  git identity: git is not installed\n")
		assertContains(t, res.Stdout, "fail  origin: git is not installed\n")
		assertContains(t, res.Stdout, "ok    ssh config:")
	})
	t.Run("ssh", func(t *testing.T) {
		t.Parallel()
		sb := doctorSandbox(t)
		sb.without("ssh")
		res := sb.run("doctor")
		assertCode(t, res, 1)
		assertContains(t, res.Stdout, "fail  ssh auth: ssh is not installed\n")
		assertContains(t, res.Stdout, "skip  workspace:")
		// Offline, the missing client does not matter.
		res = sb.run("doctor", "--offline")
		assertCode(t, res, 0)
		assertContains(t, res.Stdout, "skip  ssh auth: --offline\n")
	})
}

func TestDoctor_SummaryCountsAndExitCode(t *testing.T) {
	t.Parallel()
	sb := doctorSandbox(t)
	sb.setState(func(s *fakeState) {
		s.Git.Origin = "git@github.com:acme/app.git"
		s.Git.Local["user.name"] = gitValues{"Other"}
	})
	res := sb.run("doctor")
	assertCode(t, res, 0) // warnings alone do not fail
	assertContains(t, res.Stdout, "summary: 3 ok, 2 warn, 0 fail, 1 skip\n")

	sb.setState(func(s *fakeState) { s.SSH.Aliases["github-work"] = sshAlias{Result: "denied"} })
	res = sb.run("doctor")
	assertCode(t, res, 1)
	assertContains(t, res.Stdout, "summary: 2 ok, 2 warn, 1 fail, 1 skip\n")
	if lines := strings.Count(res.Stdout, "\n"); lines != 8 {
		t.Fatalf("%d lines, want header + 6 checks + summary", lines)
	}
}

func TestDoctor_UsageErrors(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{{"doctor", "work", "me"}, {"doctor", "--bogus"}, {"doctor", "--offline=yes"}} {
		sb := doctorSandbox(t)
		res := sb.run(args...)
		assertCode(t, res, 2)
		if len(sb.calls()) != 0 {
			t.Fatalf("%v: tools called", args)
		}
	}
	sb := doctorSandbox(t)
	res := sb.run("doctor", "nosuch")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `profile "nosuch" not found`)
}
