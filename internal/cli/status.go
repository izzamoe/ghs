package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) status() error {
	run := runner.New()
	ghStatus, ghErr := run.Output("gh", "auth", "status", "--active")
	git := gitops.New(run)
	gitName, gitEmail, gitErr := git.CurrentIdentity()
	remote, remoteErr := git.OriginURL()

	if _, err := fmt.Fprintln(a.out, "ghs status"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(a.out, "----------"); err != nil {
		return err
	}
	if err := printResult(a.out, "gh", ghStatus, ghErr); err != nil {
		return err
	}
	if gitErr != nil {
		if err := printResult(a.out, "git identity", "", gitErr); err != nil {
			return err
		}
	} else {
		if err := printResult(a.out, "git identity", gitName+" <"+gitEmail+">", nil); err != nil {
			return err
		}
	}

	return printResult(a.out, "origin", remote, remoteErr)
}

func printResult(out io.Writer, label string, value string, err error) error {
	if err != nil {
		_, writeErr := fmt.Fprintf(out, "%s: unavailable (%s)\n", label, err)

		return writeErr
	}
	value = strings.TrimSpace(value)
	if value == "" {
		value = "ok"
	}
	_, writeErr := fmt.Fprintf(out, "%s: %s\n", label, value)

	return writeErr
}
