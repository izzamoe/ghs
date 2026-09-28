package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) fixRemote(pos []string) error {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	profile, _, err := findProfile(cfg, pos[0])
	if err != nil {
		return err
	}
	git := gitops.New(runner.New())
	oldURL, _, err := git.Origin()
	if err != nil {
		return err
	}
	newURL, err := gitops.RewriteGitHubURL(oldURL, profile.SSHHostAlias, cfg.Aliases())
	if err != nil {
		return err
	}
	if err := git.SetOriginURL(newURL); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "origin updated: %s -> %s\n", oldURL, newURL)

	return err
}
