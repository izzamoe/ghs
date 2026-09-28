package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

// fixRemote rewrites origin to go through the profile's SSH alias. Only
// github.com URLs and other profiles' aliases are rewritten (FR-029/030).
func (a App) fixRemote(pos []string) error {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	profile, _, err := findProfile(cfg, pos[0])
	if err != nil {
		return err
	}
	if err := config.ValidateAlias(profile.SSHHostAlias); err != nil {
		return fmt.Errorf("profile %q: %w", profile.Name, err)
	}
	if err := requireTool("git"); err != nil {
		return err
	}
	git := gitops.New(runner.New())
	if _, inRepo, err := git.InRepo(); err != nil {
		return err
	} else if !inRepo {
		return errors.New("not inside a git repository")
	}
	url, exists, err := git.Origin()
	if err != nil {
		return err
	}
	if !exists {
		url = ""
	}
	newURL, alreadyCorrect, err := originRewrite(cfg, profile, url)
	if err != nil {
		return err
	}
	if alreadyCorrect {
		err = a.printf("origin already correct: %s", url)
	} else {
		if err := git.SetOriginURL(newURL); err != nil {
			return err
		}
		err = a.printf("origin updated: %s -> %s", url, newURL)
	}
	if err != nil {
		return err
	}

	// The rewritten remote only works once the alias exists in the SSH
	// config; say so instead of failing, because the rewrite is correct.
	if home, herr := os.UserHomeDir(); herr == nil {
		if ok, rerr := sshops.HasHostBlockInConfig(home, profile.SSHHostAlias); rerr == nil && !ok {
			a.warnf("alias %s has no block in ~/.ssh/config; run: ghs init-ssh %s", profile.SSHHostAlias, profile.Name)
		}
	}
	return nil
}
