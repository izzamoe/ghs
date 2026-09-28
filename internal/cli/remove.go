package cli

import (
	"fmt"
	"os"
	"slices"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

// remove deletes one profile section. It is reversible by construction: SSH
// keys, the SSH config, GitHub CLI logins, and Git identities are left in
// place and listed with the manual cleanup step (FR-062 to FR-066).
func (a App) remove(pos []string) error {
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	profile, idx, err := findProfile(cfg, pos[0])
	if err != nil {
		return err
	}

	// Optional, read-only: is the account still logged in or active?
	var account ghops.AuthAccount
	var accountKnown bool
	if requireTool("gh") == nil {
		if accounts, err := ghops.New(runner.New()).AuthAccounts(githubHost); err == nil {
			account = ghops.FindAccount(accounts, profile.GitHubUser)
			accountKnown = true
		}
	}

	var unlinked []string
	if profile.Workspace != "" {
		// Unlink first; if that fails the profile stays (FR-064).
		unlinked, err = unlinkWorkspaceParts(path, profile)
		if err != nil {
			return fmt.Errorf("profile %q was not removed: %w", profile.Name, err)
		}
	}
	cfg.Profiles = slices.Delete(cfg.Profiles, idx, idx+1)
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	lines := []string{fmt.Sprintf("removed profile %q from %s", profile.Name, path)}
	if profile.Workspace != "" {
		_, _, fileSlash := workspaceLinkParts(path, profile, profile.Workspace)
		detail := "nothing was linked"
		if len(unlinked) > 0 {
			detail = "removed includeIf entry and " + fileSlash
		}
		lines = append(lines, fmt.Sprintf("unlinked workspace %s (%s)", profile.Workspace, detail))
	}
	if keyPath, err := config.ExpandPath(profile.SSHKey); err == nil && profile.SSHKey != "" {
		if _, err := os.Stat(keyPath); err == nil {
			lines = append(lines, fmt.Sprintf("kept: ssh key %s (delete by hand if unused)", profile.SSHKey))
		}
	}
	if home, err := os.UserHomeDir(); err == nil && profile.SSHHostAlias != "" {
		if ok, err := sshops.HasHostBlockInConfig(home, profile.SSHHostAlias); err == nil && ok {
			lines = append(lines, fmt.Sprintf("kept: ssh config block \"Host %s\" in ~/.ssh/config (edit by hand)", profile.SSHHostAlias))
		}
	}
	if accountKnown && account.Login != "" {
		state := "logged in"
		if account.Active {
			state = "active"
		}
		lines = append(lines, fmt.Sprintf("kept: gh account %s is still %s; run: gh auth logout --hostname github.com --user %s", account.Login, state, account.Login))
	}
	for _, line := range lines {
		if err := a.printf("%s", line); err != nil {
			return err
		}
	}
	return nil
}
