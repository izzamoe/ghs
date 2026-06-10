package cli

import (
	"fmt"
	"strings"

	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

func (a App) clone(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: ghs clone <profile> <owner/repo|github-url> [directory] [--upload-key]")
	}
	profile, err := a.loadProfile(args[0])
	if err != nil {
		return err
	}
	repoInput := args[1]
	flags := parseFlags(args[2:])
	directory := cloneDirectoryArg(args[2:])
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
	if err := ssh.EnsureKey(profile); err != nil {
		return err
	}
	if err := ssh.EnsureConfig(profile); err != nil {
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

func cloneDirectoryArg(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			continue
		}
		return arg
	}
	return ""
}
