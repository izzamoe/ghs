package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

func (a App) initSSH(pos []string, flags map[string]string) error {
	profile, err := a.loadProfile(pos[0])
	if err != nil {
		return err
	}
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
	// Preflight: nothing is generated or written unless the SSH config is
	// readable and the key pair is either complete or absent.
	if _, err := sshops.ReadConfigFile(sshConfigPath); err != nil {
		return err
	}
	if _, err := sshops.CheckKeyPair(keyPath); err != nil {
		return err
	}

	ssh := sshops.New(runner.New())
	if _, created, err := ssh.EnsureKey(profile); err != nil {
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
	if hasFlagKey(flags, "upload") {
		if err := ssh.UploadKey(profile); err != nil {
			return err
		}
	}
	return a.printf("ssh is ready for profile %q via host %q", profile.Name, profile.SSHHostAlias)
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
