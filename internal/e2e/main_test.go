package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var (
	// ghsBin is the ghs binary built once for the whole run.
	ghsBin string
	// fakeSource is the directory holding one copy of the test binary per
	// fake tool name; sandboxes hard-link (or copy) from it.
	fakeSource string
)

var fakeTools = []string{"gh", "git", "ssh", "ssh-keygen"}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func TestMain(m *testing.M) {
	tool := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
	for _, name := range fakeTools {
		if tool == name {
			os.Exit(runFake(tool, os.Args[1:]))
		}
	}
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	dir, err := os.MkdirTemp("", "ghs-e2e-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: create build dir:", err)
		return 1
	}
	defer func() { _ = os.RemoveAll(dir) }()

	ghsBin = filepath.Join(dir, "ghs"+exeSuffix())
	build := exec.Command("go", "build", "-buildvcs=false", "-o", ghsBin, "github.com/izzamoe/ghs/cmd/ghs")
	build.Stdout = os.Stderr
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "e2e: build ghs:", err)
		return 1
	}

	self, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "e2e: locate test binary:", err)
		return 1
	}
	fakeSource = filepath.Join(dir, "fakes")
	if err := os.MkdirAll(fakeSource, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "e2e:", err)
		return 1
	}
	for _, name := range fakeTools {
		if err := linkOrCopy(self, filepath.Join(fakeSource, name+exeSuffix())); err != nil {
			fmt.Fprintln(os.Stderr, "e2e: install fake", name+":", err)
			return 1
		}
	}

	return m.Run()
}

// linkOrCopy hard-links src to dst, falling back to a byte copy when the two
// paths are on different file systems or links are not supported.
func linkOrCopy(src, dst string) error {
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o755)
}
