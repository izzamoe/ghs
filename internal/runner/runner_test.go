package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestMain lets the test binary act as a helper process: when
// GHS_RUNNER_HELPER is set it performs that action instead of running tests.
func TestMain(m *testing.M) {
	switch mode := os.Getenv("GHS_RUNNER_HELPER"); mode {
	case "":
		os.Exit(m.Run())
	case "pwd":
		wd, _ := os.Getwd()
		fmt.Println(wd)
		os.Exit(0)
	case "fail":
		fmt.Println("to stdout")
		fmt.Fprintln(os.Stderr, "to stderr")
		os.Exit(3)
	default:
		os.Exit(99)
	}
}

func helper(t *testing.T, mode string) string {
	t.Helper()
	t.Setenv("GHS_RUNNER_HELPER", mode)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return exe
}

func TestLookPathFindsExecutableInDir(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	name := "ghs-lookpath-probe"
	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name+suffix), data, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	got, err := New().LookPath(name)
	if err != nil {
		t.Fatalf("LookPath() error = %v", err)
	}
	if !strings.HasPrefix(got, dir) {
		t.Fatalf("LookPath() = %q, want path in %q", got, dir)
	}
	if _, err := New().LookPath("ghs-definitely-missing"); err == nil {
		t.Fatal("LookPath(missing) error = nil")
	}
}

func TestCombinedOutputReturnsOutputAndExitCode(t *testing.T) {
	exe := helper(t, "fail")

	out, code, err := New().CombinedOutput(exe)
	if err != nil {
		t.Fatalf("CombinedOutput() error = %v, want nil for a process that ran", err)
	}
	if code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
	if !strings.Contains(out, "to stdout") || !strings.Contains(out, "to stderr") {
		t.Fatalf("output = %q, want both streams", out)
	}
	if ExitCode(New().Run(exe)) != 3 {
		t.Fatal("ExitCode(Run error) != 3")
	}
	if ExitCode(nil) != 0 {
		t.Fatal("ExitCode(nil) != 0")
	}

	_, code, err = New().CombinedOutput(filepath.Join(t.TempDir(), "missing"))
	if err == nil || code != -1 {
		t.Fatalf("missing binary: code = %d, err = %v; want -1 and an error", code, err)
	}
	if _, ok := err.(*exec.ExitError); ok {
		t.Fatal("start failure must not be an ExitError")
	}
}

func TestRunInSetsWorkingDirectory(t *testing.T) {
	exe := helper(t, "pwd")
	dir := t.TempDir()

	out, err := New().OutputIn(dir, exe)
	if err != nil {
		t.Fatalf("OutputIn() error = %v", err)
	}
	gotInfo, err1 := os.Stat(strings.TrimSpace(out))
	wantInfo, err2 := os.Stat(dir)
	if err1 != nil || err2 != nil || !os.SameFile(gotInfo, wantInfo) {
		t.Fatalf("working directory = %q, want %q", out, dir)
	}
	if err := New().RunIn(dir, exe); err != nil {
		t.Fatalf("RunIn() error = %v", err)
	}
}
