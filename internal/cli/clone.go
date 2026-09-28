package cli

import (
	"fmt"
	"os"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

// clone switches to the profile's account (and stays switched, as
// documented), makes sure the key and SSH block exist, optionally uploads
// the key, clones through the alias, and sets the clone's local identity.
func (a App) clone(pos []string, flags map[string]string) error {
	profile, err := a.loadProfile(pos[0])
	if err != nil {
		return err
	}
	repoInput := pos[1]
	uploadKey := hasFlagKey(flags, "upload-key")

	// Preflight: everything that can be checked before the first mutation.
	if err := validateSSHFields(profile); err != nil {
		return err
	}
	if profile.GitEmail != "" {
		if err := validateIdentityFields(profile); err != nil {
			return err
		}
	}
	cloneURL, err := gitops.CloneURL(repoInput, profile.SSHHostAlias)
	if err != nil {
		return err
	}
	directory := gitops.CloneDirectory(repoInput)
	if len(pos) == 3 {
		directory = pos[2]
	}
	if directory == "" {
		return fmt.Errorf("clone directory could not be inferred; pass directory explicitly")
	}
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return err
	}
	sshConfigPath, err := sshops.ConfigPath()
	if err != nil {
		return err
	}
	if _, err := sshops.ReadConfigFile(sshConfigPath); err != nil {
		return err
	}
	keyExists, err := sshops.CheckKeyPair(keyPath)
	if err != nil {
		return err
	}
	if uploadKey && keyExists {
		if _, err := os.ReadFile(keyPath + ".pub"); err != nil {
			return fmt.Errorf("cannot read public key %s.pub: %w", keyPath, err)
		}
	}
	if err := requireTool("git"); err != nil {
		return err
	}
	run := runner.New()
	gh := ghops.New(run)
	active, err := accountPreflight(gh, profile)
	if err != nil {
		return err
	}

	if !equalLogin(active, profile.GitHubUser) {
		if err := gh.SwitchUser(githubHost, profile.GitHubUser); err != nil {
			return err
		}
		if err := a.printf("switched gh account: %s -> %s", displayLogin(active), profile.GitHubUser); err != nil {
			return err
		}
	}
	if _, created, err := sshops.New(run).EnsureKey(profile); err != nil {
		return err
	} else if created {
		if err := a.printf("generated ssh key %s", keyPath); err != nil {
			return err
		}
	}
	if _, added, err := sshops.EnsureConfig(profile); err != nil {
		return err
	} else if added {
		if err := a.printf("appended Host %s to %s", profile.SSHHostAlias, sshConfigPath); err != nil {
			return err
		}
	}
	if uploadKey {
		pubPath, already, err := sshops.UploadKey(gh, profile)
		if err != nil {
			return err
		}
		if already {
			err = a.printf("public key %s already registered on account %s", pubPath, profile.GitHubUser)
		} else {
			err = a.printf("uploaded public key %s to account %s", pubPath, profile.GitHubUser)
		}
		if err != nil {
			return err
		}
	}
	git := gitops.New(run)
	if err := git.Clone(cloneURL, directory); err != nil {
		return err
	}
	if profile.GitEmail != "" {
		if err := git.SetIdentityInRepo(directory, profile); err != nil {
			return err
		}
		return a.printf("cloned %s into %s and set local git identity for profile %q", cloneURL, directory, profile.Name)
	}
	return a.printf("cloned %s into %s; git identity was not set because profile email is empty", cloneURL, directory)
}

// validateIdentityFields checks the values written to Git config.
func validateIdentityFields(p config.Profile) error {
	if err := config.ValidateGitName(p.GitName); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	if err := config.ValidateEmail(p.GitEmail); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	return nil
}
