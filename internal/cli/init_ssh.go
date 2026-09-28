package cli

import (
	"fmt"
	"os"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

func (a App) initSSH(pos []string, flags map[string]string) error {
	profile, err := a.loadProfile(pos[0])
	if err != nil {
		return err
	}
	upload := hasFlagKey(flags, "upload")
	if err := validateSSHFields(profile); err != nil {
		return err
	}
	keyPath, err := config.ExpandPath(profile.SSHKey)
	if err != nil {
		return err
	}
	sshConfigPath, err := sshops.ConfigPath()
	if err != nil {
		return err
	}

	// Preflight: nothing is generated, written, or switched unless the SSH
	// config is readable, the key pair is complete or absent, and (with
	// --upload) the account is logged in and the public key is readable.
	if _, err := sshops.ReadConfigFile(sshConfigPath); err != nil {
		return err
	}
	keyExists, err := sshops.CheckKeyPair(keyPath)
	if err != nil {
		return err
	}
	run := runner.New()
	gh := ghops.New(run)
	if upload {
		if _, err := accountPreflight(gh, profile); err != nil {
			return err
		}
		if keyExists {
			if _, err := os.ReadFile(keyPath + ".pub"); err != nil {
				return fmt.Errorf("cannot read public key %s.pub: %w", keyPath, err)
			}
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
	if err := a.printf("ssh is ready for profile %q via host %q", profile.Name, profile.SSHHostAlias); err != nil {
		return err
	}
	if !upload {
		return nil
	}

	var pubPath string
	var already bool
	_, err = gh.WithAccount(githubHost, profile.GitHubUser, func() error {
		var uploadErr error
		pubPath, already, uploadErr = sshops.UploadKey(gh, profile)
		return uploadErr
	})
	if err != nil {
		return err
	}
	if already {
		return a.printf("public key %s already registered on account %s", pubPath, profile.GitHubUser)
	}
	return a.printf("uploaded public key %s to account %s", pubPath, profile.GitHubUser)
}

// validateSSHFields checks the profile values that end up in ~/.ssh/config
// (they may come from a hand-edited config file).
func validateSSHFields(p config.Profile) error {
	if err := config.ValidateAlias(p.SSHHostAlias); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	if err := config.ValidateKeyPath(p.SSHKey); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	return nil
}
