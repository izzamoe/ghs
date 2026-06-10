package runner

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type Runner struct{}

func New() Runner {
	return Runner{}
}

func (Runner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return fmt.Errorf("run %s: %w", name, err)
		}
		return fmt.Errorf("run %s: %w: %s", name, err, message)
	}
	return nil
}

func (r Runner) Output(name string, args ...string) (string, error) {
	out, err := r.OutputBytes(name, args...)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (Runner) OutputBytes(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			return nil, fmt.Errorf("run %s: %w", name, err)
		}
		return nil, fmt.Errorf("run %s: %w: %s", name, err, message)
	}
	return bytes.TrimSpace(out), nil
}
