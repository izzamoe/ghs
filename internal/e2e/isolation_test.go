package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestHarness_EnvIsMinimal(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = []ghAccount{{Login: "zamyb-work", Active: true, State: "success"}}
	})

	var path string
	for _, kv := range sb.env() {
		if v, ok := strings.CutPrefix(kv, "PATH="); ok {
			path = v
		}
	}
	if path != sb.bin {
		t.Fatalf("PATH = %q, want only the sandbox bin %q", path, sb.bin)
	}

	sb.run("status")
	calls := sb.calls()
	if len(calls) == 0 {
		t.Fatal("status invoked no fake; cannot observe the environment")
	}
	allowed := allowedEnv()
	for _, c := range calls {
		for _, name := range c.Env {
			if !slices.ContainsFunc(allowed, func(a string) bool { return strings.EqualFold(a, name) }) {
				t.Fatalf("%s saw unexpected environment variable %s (all: %v)", c, name, c.Env)
			}
		}
	}
}

func TestHarness_SandboxHomeDiffersFromRealHome(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	real, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no real home directory: %v", err)
	}
	// The sandbox may live under the real home (TMPDIR can point there), but
	// it must be a fresh directory of its own: never the real home, never
	// one of the real state directories, never a parent of the real home.
	if !strings.HasPrefix(sb.home, sb.root+string(filepath.Separator)) {
		t.Fatalf("sandbox home %q is not inside the sandbox root %q", sb.home, sb.root)
	}
	if strings.HasPrefix(real+string(filepath.Separator), sb.home+string(filepath.Separator)) {
		t.Fatalf("sandbox home %q is the real home %q or a parent of it", sb.home, real)
	}
	for _, state := range []string{".ssh", ".gitconfig", ".config", filepath.Join(".config", "gh"), filepath.Join(".config", "ghs")} {
		if strings.HasPrefix(sb.home, filepath.Join(real, state)) {
			t.Fatalf("sandbox home %q is inside the real %s", sb.home, state)
		}
	}
	for _, kv := range sb.env() {
		name, value, _ := strings.Cut(kv, "=")
		switch name {
		case "HOME", "USERPROFILE", "XDG_CONFIG_HOME", "GHS_FAKE_STATE", "GHS_FAKE_LOG", "TEMP", "TMP":
			if !strings.HasPrefix(value, sb.root) {
				t.Fatalf("%s = %q is outside the sandbox %q", name, value, sb.root)
			}
		}
	}
}

func TestHarness_LogRecordsArgsAndDir(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.runIn(sb.repoDir, "status")
	calls := sb.callsOf("git")
	if len(calls) == 0 {
		t.Fatalf("no git calls logged:\n%s", sb.logDump())
	}
	want, err := os.Stat(sb.repoDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range calls {
		got, err := os.Stat(c.Dir)
		if err != nil || !os.SameFile(got, want) {
			t.Fatalf("%s logged dir %q, want %q", c, c.Dir, sb.repoDir)
		}
		if len(c.Args) == 0 {
			t.Fatalf("%s logged no arguments", c)
		}
	}
}

func TestHarness_FakesAreFoundOnWindows(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	for _, name := range fakeTools {
		exe := filepath.Join(sb.bin, name+exeSuffix())
		if _, err := os.Stat(exe); err != nil {
			t.Fatalf("fake %s missing: %v", exe, err)
		}
		if runtime.GOOS == "windows" && !strings.HasSuffix(exe, ".exe") {
			t.Fatalf("fake %s lacks .exe on Windows", exe)
		}
	}
	sb.run("status")
	if len(sb.callsOf("git")) == 0 {
		t.Fatalf("ghs did not find the fake git on %s:\n%s", runtime.GOOS, sb.logDump())
	}
}

func TestHarness_NothingWrittenOutsideSandbox(t *testing.T) {
	t.Parallel()
	sb := newSandbox(t)
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = []ghAccount{{Login: "zamyb-work", Active: true, State: "success"}}
	})
	// t.TempDir returns a numbered directory inside a directory that belongs
	// to this test alone; watch that parent so any stray write shows up.
	parent := filepath.Dir(sb.root)
	before := sb.snapshot(parent)

	steps := [][]string{
		{"add-profile", "work", "--gh-user", "zamyb-work", "--git-name", "Izzam", "--git-email", "work@example.com", "--ssh-alias", "github-work", "--ssh-key", "~/.ssh/id_ed25519_work"},
		{"init-ssh", "work"},
		{"list"},
		{"status"},
	}
	for _, args := range steps {
		res := sb.run(args...)
		assertCode(t, res, 0)
	}

	after := sb.snapshot(parent)
	rootRel, _ := filepath.Rel(parent, sb.root)
	for path := range after {
		if _, existed := before[path]; existed && before[path] == after[path] {
			continue
		}
		if path != rootRel+"/" && !strings.HasPrefix(path, rootRel+string(filepath.Separator)) {
			t.Fatalf("%s changed outside the sandbox", path)
		}
	}
	for _, c := range sb.calls() {
		if !strings.HasPrefix(c.Dir, sb.root) {
			if resolved, err := filepath.EvalSymlinks(sb.root); err != nil || !strings.HasPrefix(c.Dir, resolved) {
				t.Fatalf("%s ran in %q, outside the sandbox", c, c.Dir)
			}
		}
	}
}
