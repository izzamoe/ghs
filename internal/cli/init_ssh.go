package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

func (a App) initSSH(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ghs init-ssh <profile> [--upload]")
	}
	profile, err := a.loadProfile(args[0])
	if err != nil {
		return err
	}
	ssh := sshops.New(runner.New())
	if err := ssh.EnsureKey(profile); err != nil {
		return err
	}
	if err := ssh.EnsureConfig(profile); err != nil {
		return err
	}
	if hasFlag(args[1:], "--upload") {
		if err := ssh.UploadKey(profile); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(a.out, "ssh is ready for profile %q via host %q\n", profile.Name, profile.SSHHostAlias)

	return err
}
