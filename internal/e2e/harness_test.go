package e2e

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sandbox is one isolated world for one test: its own HOME, config
// directory, fake tool directory, fake state, and invocation log.
type sandbox struct {
	t         *testing.T
	root      string
	home      string
	cfgHome   string
	bin       string
	tmp       string
	repoDir   string
	statePath string
	logPath   string
}

type result struct {
	Stdout string
	Stderr string
	Code   int
}

func (r result) String() string {
	return "exit=" + strconv.Itoa(r.Code) + "\n--- stdout ---\n" + r.Stdout + "--- stderr ---\n" + r.Stderr
}

func newSandbox(t *testing.T) *sandbox {
	return newSandboxHome(t, "home")
}

// newSandboxHome creates a sandbox whose home directory has the given base
// name (for example one containing a space).
func newSandboxHome(t *testing.T, homeName string) *sandbox {
	t.Helper()
	root := t.TempDir()
	sb := &sandbox{
		t:         t,
		root:      root,
		home:      filepath.Join(root, homeName),
		cfgHome:   filepath.Join(root, "config"),
		bin:       filepath.Join(root, "bin"),
		tmp:       filepath.Join(root, "tmp"),
		repoDir:   filepath.Join(root, "repo"),
		statePath: filepath.Join(root, "state.json"),
		logPath:   filepath.Join(root, "calls.log"),
	}
	for _, dir := range []string{sb.home, sb.cfgHome, sb.bin, sb.tmp, sb.repoDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range fakeTools {
		exe := name + exeSuffix()
		if err := linkOrCopy(filepath.Join(fakeSource, exe), filepath.Join(sb.bin, exe)); err != nil {
			t.Fatal(err)
		}
	}
	// Runs before t.TempDir's own cleanup. On Windows a fake started by a
	// parallel test keeps the shared (hard-linked) image mapped for a few
	// milliseconds, during which its names cannot be deleted; retry.
	t.Cleanup(func() { removeWithRetry(t, sb.bin) })
	if err := writeStateFile(sb.statePath, fakeState{Git: gitState{Repo: true, Toplevel: sb.repoDir}}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sb.logPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return sb
}

// allowedEnv is the complete set of variable names ghs may see.
func allowedEnv() []string {
	names := []string{"PATH", "HOME", "USERPROFILE", "XDG_CONFIG_HOME", "GHS_FAKE_STATE", "GHS_FAKE_LOG", "GORACE"}
	if runtime.GOOS == "windows" {
		names = append(names, "SYSTEMROOT", "TEMP", "TMP")
	}
	return names
}

func (sb *sandbox) env() []string {
	env := []string{
		"PATH=" + sb.bin,
		"HOME=" + sb.home,
		"USERPROFILE=" + sb.home,
		"XDG_CONFIG_HOME=" + sb.cfgHome,
		"GHS_FAKE_STATE=" + sb.statePath,
		"GHS_FAKE_LOG=" + sb.logPath,
		// The fakes are the race-enabled test binary; without this the race
		// runtime sleeps one second at every fake's exit.
		"GORACE=atexit_sleep_ms=0",
	}
	if runtime.GOOS == "windows" {
		env = append(env, "SYSTEMROOT="+os.Getenv("SYSTEMROOT"), "TEMP="+sb.tmp, "TMP="+sb.tmp)
	}
	return env
}

// run executes ghs in the sandbox repository directory.
func (sb *sandbox) run(args ...string) result {
	sb.t.Helper()
	return sb.runIn(sb.repoDir, args...)
}

// runIn executes ghs with the given working directory.
func (sb *sandbox) runIn(dir string, args ...string) result {
	sb.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ghsBin, args...)
	cmd.Dir = dir
	cmd.Env = sb.env()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.Code = exitErr.ExitCode()
	default:
		sb.t.Fatalf("run ghs %v: %v", args, err)
	}
	return res
}

// fake runs one fake tool directly (used by the fake self-tests).
func (sb *sandbox) fake(dir, tool string, args ...string) result {
	sb.t.Helper()
	cmd := exec.Command(filepath.Join(sb.bin, tool+exeSuffix()), args...)
	cmd.Dir = dir
	cmd.Env = sb.env()
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	switch {
	case err == nil:
	case errors.As(err, &exitErr):
		res.Code = exitErr.ExitCode()
	default:
		sb.t.Fatalf("run fake %s %v: %v", tool, args, err)
	}
	return res
}

// setState loads the fake state, lets fn change it, and saves it.
func (sb *sandbox) setState(fn func(*fakeState)) {
	sb.t.Helper()
	st := sb.state()
	fn(&st)
	if err := writeStateFile(sb.statePath, st); err != nil {
		sb.t.Fatal(err)
	}
}

func (sb *sandbox) state() fakeState {
	sb.t.Helper()
	st, err := readStateFile(sb.statePath)
	if err != nil {
		sb.t.Fatal(err)
	}
	return st
}

func (sb *sandbox) cfgDir() string     { return filepath.Join(sb.cfgHome, "ghs") }
func (sb *sandbox) cfgPath() string    { return filepath.Join(sb.cfgDir(), "config.conf") }
func (sb *sandbox) sshDir() string     { return filepath.Join(sb.home, ".ssh") }
func (sb *sandbox) sshCfgPath() string { return filepath.Join(sb.sshDir(), "config") }

// identityFile is the ghs identity file path as ghs prints and stores it.
func (sb *sandbox) identityFile(profile string) string {
	return filepath.ToSlash(filepath.Join(sb.cfgDir(), "gitconfig-"+profile))
}

func (sb *sandbox) writeConfig(content string) {
	sb.t.Helper()
	if err := os.MkdirAll(sb.cfgDir(), 0o700); err != nil {
		sb.t.Fatal(err)
	}
	if err := os.WriteFile(sb.cfgPath(), []byte(content), 0o600); err != nil {
		sb.t.Fatal(err)
	}
}

func (sb *sandbox) readConfig() string {
	sb.t.Helper()
	data, err := os.ReadFile(sb.cfgPath())
	if err != nil {
		sb.t.Fatalf("read config: %v", err)
	}
	return string(data)
}

func (sb *sandbox) configExists() bool {
	_, err := os.Stat(sb.cfgPath())
	return err == nil
}

func (sb *sandbox) writeSSHConfig(content string) {
	sb.t.Helper()
	if err := os.MkdirAll(sb.sshDir(), 0o700); err != nil {
		sb.t.Fatal(err)
	}
	if err := os.WriteFile(sb.sshCfgPath(), []byte(content), 0o600); err != nil {
		sb.t.Fatal(err)
	}
}

func (sb *sandbox) sshConfig() string {
	sb.t.Helper()
	data, err := os.ReadFile(sb.sshCfgPath())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		sb.t.Fatal(err)
	}
	return string(data)
}

// expand resolves a "~/" path inside the sandbox home.
func (sb *sandbox) expand(p string) string {
	if rest, ok := strings.CutPrefix(p, "~/"); ok {
		return filepath.Join(sb.home, filepath.FromSlash(rest))
	}
	return p
}

// touchKey creates a private key and its .pub sibling at a "~/" path.
func (sb *sandbox) touchKey(p string) string {
	sb.t.Helper()
	path := sb.expand(p)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		sb.t.Fatal(err)
	}
	base := filepath.Base(path)
	if err := os.WriteFile(path, []byte("TOUCHED PRIVATE KEY "+base+"\n"), 0o600); err != nil {
		sb.t.Fatal(err)
	}
	if err := os.WriteFile(path+".pub", []byte("ssh-ed25519 AAAATOUCH"+base+" "+base+"\n"), 0o644); err != nil {
		sb.t.Fatal(err)
	}
	return path
}

// without removes a fake tool so the sandbox models it as not installed.
func (sb *sandbox) without(tool string) {
	sb.t.Helper()
	if err := os.Remove(filepath.Join(sb.bin, tool+exeSuffix())); err != nil {
		sb.t.Fatal(err)
	}
}

func (sb *sandbox) calls() []call {
	sb.t.Helper()
	f, err := os.Open(sb.logPath)
	if err != nil {
		sb.t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []call
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			continue
		}
		var c call
		if err := json.Unmarshal(scanner.Bytes(), &c); err != nil {
			sb.t.Fatalf("parse log line %q: %v", scanner.Text(), err)
		}
		out = append(out, c)
	}
	if err := scanner.Err(); err != nil {
		sb.t.Fatal(err)
	}
	return out
}

func (sb *sandbox) callsOf(tool string) []call {
	var out []call
	for _, c := range sb.calls() {
		if c.Tool == tool {
			out = append(out, c)
		}
	}
	return out
}

// countCalls counts invocations of tool whose joined arguments contain sub.
func (sb *sandbox) countCalls(tool, sub string) int {
	n := 0
	for _, c := range sb.callsOf(tool) {
		if strings.Contains(strings.Join(c.Args, " "), sub) {
			n++
		}
	}
	return n
}

func (sb *sandbox) logDump() string {
	var b strings.Builder
	for _, c := range sb.calls() {
		b.WriteString("  " + c.String() + "\n")
	}
	return b.String()
}

// mk builds an expected call.
func mk(tool string, args ...string) call {
	return call{Tool: tool, Args: args}
}

// assertSequence checks that want appears in the log as an ordered
// subsequence (tool and exact arguments).
func (sb *sandbox) assertSequence(want ...call) {
	sb.t.Helper()
	got := sb.calls()
	i := 0
	for _, c := range got {
		if i < len(want) && c.Tool == want[i].Tool && slices.Equal(c.Args, want[i].Args) {
			i++
		}
	}
	if i < len(want) {
		var w strings.Builder
		for _, c := range want {
			w.WriteString("  " + c.String() + "\n")
		}
		sb.t.Fatalf("call sequence not found (matched %d of %d)\nwant subsequence:\n%sgot:\n%s", i, len(want), w.String(), sb.logDump())
	}
}

// isMutating reports whether a logged call changes state, per the list in
// contracts/fake-tools.md.
func isMutating(c call) bool {
	args := c.Args
	switch c.Tool {
	case "gh":
		if len(args) >= 2 {
			switch args[0] + " " + args[1] {
			case "auth switch", "auth logout", "ssh-key add":
				return true
			}
		}
		return false
	case "git":
		if len(args) >= 2 && args[0] == "-C" {
			args = args[2:]
		}
		if len(args) == 0 {
			return false
		}
		switch args[0] {
		case "clone":
			return true
		case "remote":
			return len(args) >= 2 && args[1] == "set-url"
		case "config":
			for _, a := range args[1:] {
				switch a {
				case "--get", "--get-all", "--show-origin", "--list":
					return false
				}
			}
			return true
		}
		return false
	case "ssh-keygen":
		return true
	}
	return false
}

func (sb *sandbox) assertNoMutations() {
	sb.t.Helper()
	for _, c := range sb.calls() {
		if isMutating(c) {
			sb.t.Fatalf("unexpected mutating call %s\nlog:\n%s", c, sb.logDump())
		}
	}
}

// snapshot maps every file under dir (a "~/" path or absolute) to its bytes.
func (sb *sandbox) snapshot(dir string) map[string]string {
	sb.t.Helper()
	root := sb.expand(dir)
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if d.IsDir() {
			out[rel+"/"] = ""
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = string(data)
		return nil
	})
	if err != nil {
		sb.t.Fatal(err)
	}
	return out
}

func assertSnapshotEqual(t *testing.T, before, after map[string]string) {
	t.Helper()
	for k, v := range before {
		if got, ok := after[k]; !ok {
			t.Errorf("%s disappeared", k)
		} else if got != v {
			t.Errorf("%s changed:\nbefore: %q\nafter:  %q", k, v, got)
		}
	}
	for k := range after {
		if _, ok := before[k]; !ok {
			t.Errorf("%s appeared", k)
		}
	}
}

func assertCode(t *testing.T, res result, want int) {
	t.Helper()
	if res.Code != want {
		t.Fatalf("exit code = %d, want %d\n%s", res.Code, want, res)
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("output does not contain %q:\n%s", want, got)
	}
}

func assertNotContains(t *testing.T, got, unwanted string) {
	t.Helper()
	if strings.Contains(got, unwanted) {
		t.Fatalf("output unexpectedly contains %q:\n%s", unwanted, got)
	}
}

func skipOnWindows(t *testing.T, reason string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip(reason)
	}
}

// skipUnlessPermissionsEnforced skips tests that rely on chmod 000 denying
// reads (not true on Windows or for root).
func skipUnlessPermissionsEnforced(t *testing.T) {
	t.Helper()
	skipOnWindows(t, "file mode bits do not deny reads on Windows")
	if os.Geteuid() == 0 {
		t.Skip("running as root: mode 000 does not deny reads")
	}
}

func removeWithRetry(t *testing.T, dir string) {
	deadline := time.Now().Add(30 * time.Second)
	for {
		err := os.RemoveAll(dir)
		if err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Errorf("remove %s: %v", dir, err)
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
}
