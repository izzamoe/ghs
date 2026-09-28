package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) useProfile(pos []string, flags map[string]string) error {
	profile, err := a.loadProfile(pos[0])
	if err != nil {
		return err
	}
	run := runner.New()
	if err := run.Run("gh", "auth", "switch", "--hostname", "github.com", "--user", profile.GitHubUser); err != nil {
		return err
	}
	if profile.GitEmail == "" {
		_, err = fmt.Fprintf(a.out, "using profile %q for gh only; git email is empty, so git identity was not changed\n", profile.Name)

		return err
	}
	if _, err := gitops.New(run).SetIdentity(profile, hasFlagKey(flags, "global")); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "using profile %q for gh and git identity\n", profile.Name)

	return err
}
