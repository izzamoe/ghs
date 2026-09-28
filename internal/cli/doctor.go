package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
	"github.com/izzamoe/ghs/internal/sshops"
)

const (
	outcomeOK   = "ok"
	outcomeWarn = "warn"
	outcomeFail = "fail"
	outcomeSkip = "skip"
)

// doctorCheck is one line of doctor output.
type doctorCheck struct {
	Name    string
	Outcome string
	Message string
}

func check(name, outcome, format string, args ...any) doctorCheck {
	return doctorCheck{Name: name, Outcome: outcome, Message: fmt.Sprintf(format, args...)}
}

// doctor runs six read-only checks in a fixed order and exits 1 when any
// fails (FR-067 to FR-076). It never writes a file and never calls a
// mutating tool operation.
func (a App) doctor(pos []string, flags map[string]string) error {
	cfgPath, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	var profile config.Profile
	selected := false
	if len(pos) == 1 {
		profile, _, err = findProfile(cfg, pos[0])
		if err != nil {
			return err
		}
		selected = true
	}

	checks := []doctorCheck{doctorGH(cfg, &profile, &selected)}
	if !selected {
		for _, name := range []string{"git identity", "origin", "ssh config", "ssh auth", "workspace"} {
			checks = append(checks, check(name, outcomeSkip, "no profile selected"))
		}
	} else {
		run := runner.New()
		git := gitops.New(run)
		haveGit := requireTool("git") == nil
		var inRepo bool
		var repoErr error
		if haveGit {
			_, inRepo, repoErr = git.InRepo()
		}
		checks = append(checks,
			doctorIdentity(git, profile, haveGit, inRepo, repoErr),
			doctorOrigin(git, cfg, profile, haveGit, inRepo, repoErr),
			doctorSSHConfig(profile),
			doctorSSHAuth(run, profile, hasFlagKey(flags, "offline")),
			doctorWorkspace(git, cfgPath, profile, haveGit),
		)
	}

	header := "ghs doctor: no profile selected"
	if selected {
		header = fmt.Sprintf("ghs doctor: profile %q", profile.Name)
	}
	if err := a.printf("%s", header); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, c := range checks {
		counts[c.Outcome]++
		if err := a.printf("%-6s%s: %s", c.Outcome, c.Name, c.Message); err != nil {
			return err
		}
	}
	if err := a.printf("summary: %d ok, %d warn, %d fail, %d skip", counts[outcomeOK], counts[outcomeWarn], counts[outcomeFail], counts[outcomeSkip]); err != nil {
		return err
	}
	if n := counts[outcomeFail]; n > 0 {
		noun := "checks"
		if n == 1 {
			noun = "check"
		}
		return fmt.Errorf("doctor found %d failing %s", n, noun)
	}
	return nil
}

// doctorGH checks the GitHub CLI and, without an explicit profile, selects
// the profile whose gh_user is the active account (FR-069, FR-070).
func doctorGH(cfg config.Config, profile *config.Profile, selected *bool) doctorCheck {
	const name = "gh"
	view := readAccounts()
	switch {
	case !view.known && view.reason == "gh not found":
		return check(name, outcomeFail, "%v", errGHMissing)
	case !view.known:
		return check(name, outcomeFail, "cannot read auth status: %s", view.reason)
	case view.active == "":
		return check(name, outcomeFail, "no active account for github.com; run: gh auth login")
	}
	if !*selected {
		p, ok := cfg.ByLogin(view.active)
		if !ok {
			return check(name, outcomeFail, "active account %s matches no profile; run: ghs use <profile>", view.active)
		}
		*profile, *selected = p, true
	}
	if !strings.EqualFold(view.active, profile.GitHubUser) {
		return check(name, outcomeFail, "active account %s is not profile %q (%s); run: ghs use %s", view.active, profile.Name, profile.GitHubUser, profile.Name)
	}
	return check(name, outcomeOK, "active account %s (profile %q)", view.active, profile.Name)
}

func doctorIdentity(git gitops.Git, p config.Profile, haveGit, inRepo bool, repoErr error) doctorCheck {
	const name = "git identity"
	switch {
	case !haveGit:
		return check(name, outcomeFail, "git is not installed")
	case repoErr != nil:
		return check(name, outcomeFail, "cannot run git: %s", shortReason(repoErr))
	}
	id, err := git.IdentityWithOrigin(!inRepo)
	if err != nil {
		return check(name, outcomeFail, "cannot read git config: %s", shortReason(err))
	}
	scope := scopeWord(id.EmailScope, inRepo)
	switch {
	case id.Email == "":
		return check(name, outcomeFail, "user.email is not set (%s); run: ghs use %s", scopeWord("", inRepo), p.Name)
	case p.GitEmail == "":
		return check(name, outcomeFail, "profile %q has no email; run: ghs set-email %s <email>", p.Name, p.Name)
	case !strings.EqualFold(id.Email, p.GitEmail):
		return check(name, outcomeFail, "email %s differs from profile %s (%s, %s); run: ghs use %s", id.Email, p.GitEmail, scope, id.EmailFile, p.Name)
	case id.Name != p.GitName:
		return check(name, outcomeWarn, "name %q differs from profile %q (%s, %s)", id.Name, p.GitName, scopeWord(id.NameScope, inRepo), id.NameFile)
	}
	return check(name, outcomeOK, "%s <%s> (%s, %s)", id.Name, id.Email, scope, id.EmailFile)
}

func doctorOrigin(git gitops.Git, cfg config.Config, p config.Profile, haveGit, inRepo bool, repoErr error) doctorCheck {
	const name = "origin"
	switch {
	case !haveGit:
		return check(name, outcomeFail, "git is not installed")
	case repoErr != nil:
		return check(name, outcomeFail, "cannot run git: %s", shortReason(repoErr))
	case !inRepo:
		return check(name, outcomeSkip, "not inside a git repository")
	}
	url, exists, err := git.Origin()
	switch {
	case err != nil:
		return check(name, outcomeFail, "cannot read origin: %s", shortReason(err))
	case !exists:
		return check(name, outcomeSkip, "no origin remote")
	}
	res, remote := resolveOrigin(cfg, url)
	switch res.Kind {
	case kindUnaliased:
		return check(name, outcomeWarn, "%s uses github.com directly; run: ghs fix-remote %s", url, p.Name)
	case kindProfile:
		if res.Profile == p.Name {
			return check(name, outcomeOK, "%s uses alias %s", url, p.SSHHostAlias)
		}
		return check(name, outcomeFail, "%s uses alias of profile %q; run: ghs fix-remote %s", url, res.Profile, p.Name)
	case kindUnknownAlias:
		return check(name, outcomeFail, "%s uses unknown alias %s; run: ghs fix-remote %s", url, remote.Host, p.Name)
	}
	host := remote.Host
	if host == "" {
		host = "unknown"
	}
	return check(name, outcomeSkip, "%s is not a GitHub remote (host %s)", url, host)
}

func doctorSSHConfig(p config.Profile) doctorCheck {
	const name = "ssh config"
	home, err := os.UserHomeDir()
	if err != nil {
		return check(name, outcomeFail, "cannot resolve home directory: %v", err)
	}
	content, path, err := sshops.ReadConfig(home)
	if err != nil {
		reason := err.Error()
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			reason = pathErr.Err.Error()
		}
		return check(name, outcomeFail, "cannot read %s: %s", path, reason)
	}
	remedy := "run: ghs init-ssh " + p.Name
	block := sshops.ParseHostBlock(content, p.SSHHostAlias)
	switch {
	case !block.Found:
		return check(name, outcomeFail, "no Host block for %s in %s; %s", p.SSHHostAlias, path, remedy)
	case !strings.EqualFold(block.HostName, "github.com"):
		return check(name, outcomeFail, "Host %s is missing HostName github.com; %s", p.SSHHostAlias, remedy)
	case block.User != "git":
		return check(name, outcomeFail, "Host %s is missing User git; %s", p.SSHHostAlias, remedy)
	case !strings.EqualFold(block.IdentitiesOnly, "yes"):
		return check(name, outcomeFail, "Host %s is missing IdentitiesOnly yes; %s", p.SSHHostAlias, remedy)
	case block.IdentityFile == "":
		return check(name, outcomeFail, "Host %s is missing IdentityFile %s; %s", p.SSHHostAlias, p.SSHKey, remedy)
	}
	if _, err := os.Stat(block.IdentityFile); err != nil {
		return check(name, outcomeFail, "IdentityFile %s does not exist; %s", block.IdentityFile, remedy)
	}
	if _, err := os.Stat(block.IdentityFile + ".pub"); err != nil {
		return check(name, outcomeFail, "IdentityFile %s has no %s.pub; %s", block.IdentityFile, block.IdentityFile, remedy)
	}
	return check(name, outcomeOK, "Host %s -> github.com, IdentityFile %s", p.SSHHostAlias, block.IdentityFile)
}

func doctorSSHAuth(run runner.Runner, p config.Profile, offline bool) doctorCheck {
	const name = "ssh auth"
	if offline {
		return check(name, outcomeSkip, "--offline")
	}
	if err := requireTool("ssh"); err != nil {
		return check(name, outcomeFail, "ssh is not installed")
	}
	res, err := sshops.Verify(run, p.SSHHostAlias)
	if err != nil {
		return check(name, outcomeFail, "cannot run ssh: %v", err)
	}
	if res.Kind == sshops.AuthLogin {
		if strings.EqualFold(res.Login, p.GitHubUser) {
			return check(name, outcomeOK, "github authenticated %s as %s", p.SSHHostAlias, res.Login)
		}
		return check(name, outcomeFail, "github authenticated %s as %s, expected %s", p.SSHHostAlias, res.Login, p.GitHubUser)
	}
	return check(name, outcomeFail, "%s; run once: ssh -T git@%s", res.Message, p.SSHHostAlias)
}

func doctorWorkspace(git gitops.Git, cfgPath string, p config.Profile, haveGit bool) doctorCheck {
	const name = "workspace"
	if p.Workspace == "" {
		return check(name, outcomeSkip, "profile has no workspace")
	}
	if !haveGit {
		return check(name, outcomeFail, "git is not installed")
	}
	remedy := fmt.Sprintf("run: ghs workspace %s %s", p.Name, p.Workspace)
	key, file, fileSlash := workspaceLinkParts(cfgPath, p, config.NormalizeWorkspace(p.Workspace))
	hasEntry, err := git.HasInclude(key, fileSlash)
	if err != nil {
		return check(name, outcomeFail, "cannot read global git config: %s", shortReason(err))
	}
	if !hasEntry {
		return check(name, outcomeFail, "%s is configured but not linked; %s", p.Workspace, remedy)
	}
	gotName, gotEmail, err := config.ReadIdentityFile(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return check(name, outcomeFail, "identity file %s is missing; %s", fileSlash, remedy)
	case err != nil:
		return check(name, outcomeFail, "cannot read identity file %s: %v", fileSlash, err)
	case gotName != p.GitName || !strings.EqualFold(gotEmail, p.GitEmail):
		return check(name, outcomeFail, "identity file %s differs from profile; %s", fileSlash, remedy)
	}
	return check(name, outcomeOK, "%s linked via %s", p.Workspace, fileSlash)
}
