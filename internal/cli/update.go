package cli

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) update() error {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Path == "" || info.Path == "command-line-arguments" {
		return fmt.Errorf("update is only available when installed via go install")
	}

	current := info.Main.Version
	if current == "" || current == "(devel)" {
		return fmt.Errorf("update is not available for local builds; run: go install %s@latest", info.Path)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate binary: %w", err)
	}

	_, _ = fmt.Fprintln(a.out, "updating ghs...")
	run := runner.New()
	if err := run.Run("go", "install", info.Path+"@latest"); err != nil {
		return err
	}

	next, err := run.Output(exe, "version")
	if err != nil {
		_, _ = fmt.Fprintf(a.out, "updated from %s\n", current)
		return nil
	}
	next = strings.TrimPrefix(strings.TrimSpace(next), "ghs ")

	if next == current {
		_, err = fmt.Fprintf(a.out, "ghs %s is already up to date\n", current)
	} else {
		_, err = fmt.Fprintf(a.out, "updated ghs %s → %s\n", current, next)
	}
	return err
}
