package e2e

import (
	"slices"
	"strings"
	"testing"
)

// cliCommands mirrors the command table: minimal valid positionals and the
// accepted flags, so every command is exercised by the flag tests.
var cliCommands = []struct {
	name  string
	pos   []string
	flags []string
}{
	{"add-profile", []string{"work"}, []string{"--gh-user", "--git-name", "--git-email", "--ssh-alias", "--ssh-key", "--workspace"}},
	{"add-from-gh", []string{"work"}, []string{"--git-name", "--git-email", "--ssh-alias", "--ssh-key", "--workspace", "--require-email"}},
	{"import-all", nil, []string{"--hostname", "--require-email", "--no-overwrite"}},
	{"list", nil, nil},
	{"status", nil, nil},
	{"doctor", nil, []string{"--offline"}},
	{"set-email", []string{"work", "a@example.com"}, nil},
	{"workspace", []string{"work", "~/Documents/work"}, []string{"--unlink"}},
	{"use", []string{"work"}, []string{"--global", "--fix-remote"}},
	{"clone", []string{"work", "acme/app"}, []string{"--upload-key"}},
	{"init-ssh", []string{"work"}, []string{"--upload"}},
	{"fix-remote", []string{"work"}, nil},
	{"remove", []string{"work"}, nil},
	{"version", nil, nil},
	{"update", nil, nil},
}

func assertUsageError(t *testing.T, sb *sandbox, res result, message string) {
	t.Helper()
	assertCode(t, res, 2)
	if !strings.HasPrefix(res.Stderr, "ghs: ") {
		t.Fatalf("stderr = %q, want ghs: prefix", res.Stderr)
	}
	assertContains(t, res.Stderr, message)
	if res.Stdout != "" {
		t.Fatalf("stdout on usage error = %q", res.Stdout)
	}
	if n := len(sb.calls()); n != 0 {
		t.Fatalf("%d tool calls on a usage error:\n%s", n, sb.logDump())
	}
}

func TestFlags_UnknownFlagEveryCommand(t *testing.T) {
	t.Parallel()
	if len(cliCommands) != 15 {
		t.Fatalf("table covers %d commands, want 15", len(cliCommands))
	}
	for _, c := range cliCommands {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			sb := newSandbox(t)
			sb.writeConfig(cfgAll)
			args := append(append([]string{c.name}, c.pos...), "--bogus")
			res := sb.run(args...)
			assertUsageError(t, sb, res, "unknown flag --bogus")
			if len(c.flags) == 0 {
				assertContains(t, res.Stderr, "takes no flags")
			}
			for _, f := range c.flags {
				assertContains(t, res.Stderr, f)
			}
			assertContains(t, res.Stderr, "ghs "+c.name)
		})
	}
}

func TestFlags_MissingValueAtEndAndBeforeFlag(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"add-profile", "work", "--gh-user"},
		{"add-profile", "work", "--gh-user", "--git-name", "Izzam"},
		{"add-from-gh", "work", "--git-email"},
		{"import-all", "--hostname"},
		{"add-profile", "work", "--ssh-key="},
	} {
		sb := newSandbox(t)
		assertUsageError(t, sb, sb.run(args...), "requires a value")
	}
}

func TestFlags_BooleanWithValue(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"use", "work", "--global", "yes"},
		{"use", "--global", "yes", "work"},
		{"use", "work", "--global=yes"},
		{"init-ssh", "work", "--upload=true"},
		{"import-all", "--no-overwrite", "please"},
	} {
		sb := newSandbox(t)
		sb.writeConfig(cfgAll)
		assertUsageError(t, sb, sb.run(args...), "does not take a value")
	}
}

func TestFlags_DuplicateFlag(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"use", "work", "--global", "--global"},
		append(addWorkArgs(), "--gh-user", "other"),
		{"doctor", "--offline", "--offline"},
	} {
		sb := newSandbox(t)
		sb.writeConfig(cfgAll)
		assertUsageError(t, sb, sb.run(args...), "given more than once")
	}
}

func TestFlags_SurplusPositional(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"list", "extra"},
		{"status", "extra"},
		{"version", "extra"},
		{"update", "extra"},
		{"use", "work", "extra"},
		{"init-ssh", "work", "extra"},
		{"fix-remote", "work", "extra"},
		{"set-email", "work", "a@example.com", "extra"},
		{"remove", "work", "extra"},
		{"doctor", "work", "extra"},
		{"workspace", "work", "~/a", "extra"},
		{"clone", "work", "acme/app", "dir", "extra"},
		{"import-all", "extra"},
	} {
		sb := newSandbox(t)
		sb.writeConfig(cfgAll)
		assertUsageError(t, sb, sb.run(args...), `unexpected argument "extra"`)
	}
	// Too few positionals are usage errors too.
	for _, args := range [][]string{{"use"}, {"set-email", "work"}, {"clone", "work"}, {"remove"}} {
		sb := newSandbox(t)
		assertUsageError(t, sb, sb.run(args...), "missing required argument")
	}
}

func TestFlags_BeforePositionalEquivalent(t *testing.T) {
	t.Parallel()
	logFor := func(args ...string) ([]string, result) {
		sb := newSandbox(t)
		sb.seedStandard()
		res := sb.run(args...)
		var out []string
		for _, c := range sb.calls() {
			out = append(out, c.String())
		}
		return out, res
	}
	before, r1 := logFor("use", "--global", "me")
	after, r2 := logFor("use", "me", "--global")
	assertCode(t, r1, 0)
	assertCode(t, r2, 0)
	if !slices.Equal(before, after) || r1.Stdout != r2.Stdout {
		t.Fatalf("flag position changed behavior:\n%v\n%v", before, after)
	}
}

func TestFlags_DoubleDashEndsFlags(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)

	// After --, "--global" is the profile name, not a flag.
	res := sb.run("use", "--", "--global")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `profile "--global" not found`)

	res = sb.run("set-email", "work", "--", "new@example.com")
	assertCode(t, res, 0)
	assertContains(t, sb.readConfig(), `git_email = "new@example.com"`)
}

func TestFlags_UsageErrorNoSideEffects(t *testing.T) {
	t.Parallel()
	for _, args := range [][]string{
		{"use", "work", "--globl"},
		{"add-profile", "work"},
		{"init-ssh", "work", "--upload", "extra"},
		{"clone", "work", "acme/app", "--upload"},
		{"workspace", "work"},
		{"workspace", "work", "~/Documents/work", "--unlink"},
		{"import-all", "--hostname", "ghe.example.com"},
	} {
		sb := newSandbox(t)
		sb.seedAccounts("zamyb", "zamyb", "zamyb-work")
		res := sb.run(args...)
		assertCode(t, res, 2)
		if n := len(sb.calls()); n != 0 {
			t.Fatalf("%v: %d tool calls:\n%s", args, n, sb.logDump())
		}
		if sb.configExists() || sb.sshConfig() != "" {
			t.Fatalf("%v: files written", args)
		}
	}
}

func TestFlags_HelpPerCommandStdoutExit0(t *testing.T) {
	t.Parallel()
	for _, c := range cliCommands {
		for _, flag := range []string{"--help", "-h"} {
			sb := newSandbox(t)
			args := append(append([]string{c.name}, c.pos...), flag)
			res := sb.run(args...)
			assertCode(t, res, 0)
			if !strings.HasPrefix(res.Stdout, "ghs "+c.name) {
				t.Fatalf("%v: stdout = %q", args, res.Stdout)
			}
			for _, f := range c.flags {
				assertContains(t, res.Stdout, f)
			}
			if res.Stderr != "" || len(sb.calls()) != 0 {
				t.Fatalf("%v: stderr %q, calls %d", args, res.Stderr, len(sb.calls()))
			}
		}
	}
	sb := newSandbox(t)
	res := sb.run("--help")
	assertCode(t, res, 0)
	assertContains(t, res.Stdout, "Only github.com is supported.")
}

func TestFlags_UnknownCommandExit2(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	res := sb.run("frobnicate", "--x")
	assertUsageError(t, sb, res, `unknown command "frobnicate"`)
	assertContains(t, res.Stderr, "ghs add-profile <name>")
}

func TestFlags_RuntimeFailureExit1(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.writeConfig(cfgAll)
	res := sb.run("use", "nosuch")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `ghs: profile "nosuch" not found`)

	res = sb.run("use", "WORK")
	assertCode(t, res, 1)
	assertContains(t, res.Stderr, `did you mean "work"?`)
}
