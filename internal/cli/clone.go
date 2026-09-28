package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

func (a App) clone(pos []string, flags map[string]string) error {
	profile, err := a.loadProfile(pos[0])
	if err != nil {
		return err
	}
	repoInput := pos[1]
	directory := ""
	if len(pos) == 3 {
		directory = pos[2]
	}
	if directory == "" {
		directory = gitops.CloneDirectory(repoInput)
	}
	if directory == "" {
		return fmt.Errorf("clone directory could not be inferred; pass directory explicitly")
	}
	cloneURL, err := gitops.CloneURL(repoInput, profile.SSHHostAlias)
	if err != nil {
		return err
	}

	run := runner.New()
	if err := run.Run("gh", "auth", "switch", "--hostname", "github.com", "--user", profile.GitHubUser); err != nil {
		return err
	}
	ssh := sshops.New(run)
	if _, _, err := ssh.EnsureKey(profile); err != nil {
		return err
	}
	if _, _, err := sshops.EnsureConfig(profile); err != nil {
		return err
	}
	if hasFlagKey(flags, "upload-key") {
		if err := ssh.UploadKey(profile); err != nil {
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
		_, err = fmt.Fprintf(a.out, "cloned %s into %s and set local git identity for profile %q\n", cloneURL, directory, profile.Name)

		return err
	}
	_, err = fmt.Fprintf(a.out, "cloned %s into %s; git identity was not set because profile email is empty\n", cloneURL, directory)

	return err
}
