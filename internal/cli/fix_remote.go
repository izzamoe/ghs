package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) fixRemote(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ghs fix-remote <profile>")
	}
	profile, err := a.loadProfile(args[0])
	if err != nil {
		return err
	}
	git := gitops.New(runner.New())
	oldURL, err := git.OriginURL()
	if err != nil {
		return err
	}
	newURL, err := gitops.RewriteGitHubURL(oldURL, profile.SSHHostAlias)
	if err != nil {
		return err
	}
	if err := git.SetOriginURL(newURL); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "origin updated: %s -> %s\n", oldURL, newURL)

	return err
}
