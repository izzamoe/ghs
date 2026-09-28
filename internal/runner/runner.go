package runner

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes external tools with an argument vector, never through a
// shell. The zero value runs in the current working directory.
type Runner struct{}

func New() Runner {
	return Runner{}
}

// LookPath reports where name would be found on PATH.
func (Runner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r Runner) Run(name string, args ...string) error {
	return r.RunIn("", name, args...)
}

// RunIn runs name in dir ("" = current directory), discarding stdout.
func (Runner) RunIn(dir string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	return wrap(name, cmd.Run(), stderr.String())
}

func (r Runner) Output(name string, args ...string) (string, error) {
	return r.OutputIn("", name, args...)
}

// OutputIn runs name in dir and returns its trimmed stdout.
func (r Runner) OutputIn(dir string, name string, args ...string) (string, error) {
	out, err := r.outputBytes(dir, name, args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (r Runner) OutputBytes(name string, args ...string) ([]byte, error) {
	return r.outputBytes("", name, args...)
}

func (Runner) outputBytes(dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, wrap(name, err, stderr.String())
	}
	return bytes.TrimSpace(out), nil
}

// CombinedOutput runs name and returns stdout and stderr interleaved plus the
// exit code. A process that ran and exited non-zero is not an error; err is
// non-nil (and code -1) only when the process could not be started.
func (Runner) CombinedOutput(name string, args ...string) (string, int, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return string(out), 0, nil
	case errors.As(err, &exitErr):
		return string(out), exitErr.ExitCode(), nil
	default:
		return string(out), -1, fmt.Errorf("run %s: %w", name, err)
	}
}

// ExitCode extracts the exit status from an error returned by Run, RunIn,
// Output, OutputIn, or OutputBytes: 0 for nil, -1 when the process did not
// run. Any error in the chain with an ExitCode() int method (such as
// *exec.ExitError) counts, so test stubs can carry exit codes too.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return -1
}

func wrap(name string, err error, stderr string) error {
	if err == nil {
		return nil
	}
	message := strings.TrimSpace(stderr)
	if message == "" {
		return fmt.Errorf("run %s: %w", name, err)
	}
	return fmt.Errorf("run %s: %w: %s", name, err, message)
}
