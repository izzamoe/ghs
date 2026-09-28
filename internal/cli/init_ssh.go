package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

func (a App) initSSH(pos []string, flags map[string]string) error {
	profile, err := a.loadProfile(pos[0])
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
	if hasFlagKey(flags, "upload") {
		if err := ssh.UploadKey(profile); err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(a.out, "ssh is ready for profile %q via host %q\n", profile.Name, profile.SSHHostAlias)

	return err
}
