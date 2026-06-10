package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) useProfile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ghs use <profile> [--global]")
	}
	profile, err := a.loadProfile(args[0])
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
	if err := gitops.New(run).SetIdentity(profile, hasFlag(args[1:], "--global")); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "using profile %q for gh and git identity\n", profile.Name)

	return err
}
