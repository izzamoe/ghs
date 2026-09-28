package cli

import (
	"errors"
	"fmt"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

// usePlan is everything ghs use decides during preflight, before its first
// mutation.
type usePlan struct {
	cfg       config.Config
	profile   config.Profile
	global    bool
	fixRemote bool
	haveGit   bool
	inRepo    bool
	toplevel  string
	previous  string // active login before the command ("" = none)

	originOld     string // with --fix-remote: current origin
	originNew     string // with --fix-remote: rewritten origin
	originCorrect bool   // with --fix-remote: origin already uses the alias
}

func (a App) useProfile(pos []string, flags map[string]string) error {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	profile, _, err := findProfile(cfg, pos[0])
	if err != nil {
		return err
	}
	run := runner.New()
	gh := ghops.New(run)
	git := gitops.New(run)

	plan, err := preflightUse(gh, git, cfg, profile, hasFlagKey(flags, "global"), hasFlagKey(flags, "fix-remote"))
	if err != nil {
		return err
	}
	lines, err := applyUse(gh, git, plan)
	if err != nil {
		return err
	}
	for _, line := range lines {
		if err := a.printf("%s", line); err != nil {
			return err
		}
	}

	// After a successful switch, a read-only look at origin tells the user
	// when pushes would still bypass the profile's key.
	if !plan.fixRemote && plan.inRepo {
		if url, exists, err := git.Origin(); err == nil && exists {
			if warning := originWarning(cfg, profile, url); warning != "" {
				a.warnf("%s", warning)
			}
		}
	}
	return nil
}

// preflightUse checks every precondition before anything changes (FR-026,
// FR-050): the profile's values, the account's authentication, the
// identity target, and with --fix-remote the rewrite itself.
func preflightUse(gh ghops.GH, git gitops.Git, cfg config.Config, profile config.Profile, global, fixRemote bool) (usePlan, error) {
	plan := usePlan{cfg: cfg, profile: profile, global: global, fixRemote: fixRemote}
	if profile.GitEmail != "" {
		if err := validateIdentityFields(profile); err != nil {
			return plan, err
		}
	}
	if fixRemote {
		if err := config.ValidateAlias(profile.SSHHostAlias); err != nil {
			return plan, fmt.Errorf("profile %q: %w", profile.Name, err)
		}
	}
	previous, err := accountPreflight(gh, profile)
	if err != nil {
		return plan, err
	}
	plan.previous = previous

	gitErr := requireTool("git")
	plan.haveGit = gitErr == nil
	if !plan.haveGit && (profile.GitEmail != "" || fixRemote) {
		return plan, gitErr
	}
	if plan.haveGit {
		plan.toplevel, plan.inRepo, err = git.InRepo()
		if err != nil {
			return plan, err
		}
	}
	if fixRemote {
		if !plan.inRepo {
			return plan, errors.New("--fix-remote needs a repository: not inside a git repository")
		}
		url, exists, err := git.Origin()
		if err != nil {
			return plan, err
		}
		if !exists {
			url = ""
		}
		plan.originOld = url
		plan.originNew, plan.originCorrect, err = originRewrite(cfg, profile, url)
		if err != nil {
			return plan, err
		}
	}
	if profile.GitEmail != "" && !global && !plan.inRepo {
		return plan, errors.New("not inside a git repository; pass --global to set the global identity")
	}
	return plan, nil
}

// applyUse performs the mutations in order (origin, account, identity) and,
// on failure, undoes the earlier ones in reverse order, reporting every
// result (FR-027, FR-051).
func applyUse(gh ghops.GH, git gitops.Git, plan usePlan) ([]string, error) {
	p := plan.profile
	var lines []string
	var undo []func() error

	rollback := func(cause error) error {
		errs := []error{cause}
		for i := len(undo) - 1; i >= 0; i-- {
			errs = append(errs, undo[i]())
		}
		return errors.Join(errs...)
	}

	if plan.fixRemote {
		if plan.originCorrect {
			lines = append(lines, "origin already correct: "+plan.originOld)
		} else {
			if err := git.SetOriginURL(plan.originNew); err != nil {
				return nil, rollback(fmt.Errorf("rewrite origin: %w", err))
			}
			lines = append(lines, fmt.Sprintf("origin updated: %s -> %s", plan.originOld, plan.originNew))
			old := plan.originOld
			undo = append(undo, func() error {
				if err := git.SetOriginURL(old); err != nil {
					return fmt.Errorf("could not restore origin to %s: %w", old, err)
				}
				return fmt.Errorf("restored origin to %s", old)
			})
		}
	}

	if equalLogin(plan.previous, p.GitHubUser) {
		lines = append(lines, "gh account already active: "+p.GitHubUser)
	} else {
		if err := gh.SwitchUser(githubHost, p.GitHubUser); err != nil {
			return nil, rollback(fmt.Errorf("switch gh account to %s: %w", p.GitHubUser, err))
		}
		lines = append(lines, fmt.Sprintf("switched gh account: %s -> %s", displayLogin(plan.previous), p.GitHubUser))
		if previous := plan.previous; previous != "" {
			undo = append(undo, func() error { return restoreAccount(gh, previous) })
		}
	}

	if p.GitEmail == "" {
		lines = append(lines, fmt.Sprintf("git identity unchanged: profile %q has no email; run: ghs set-email %s <email>", p.Name, p.Name))
		return lines, nil
	}
	scope := "global"
	if !plan.global {
		scope = "local: " + plan.toplevel
	}
	if step, err := git.SetIdentity(p, plan.global); err != nil {
		cause := fmt.Errorf("set git %s: %w", step, err)
		if step == "user.email" {
			cause = errors.Join(cause, fmt.Errorf("note: git user.name was already set to %q (%s)", p.GitName, scope))
		}
		return nil, rollback(cause)
	}
	lines = append(lines, fmt.Sprintf("git identity set: %s <%s> (%s)", p.GitName, p.GitEmail, scope))
	return lines, nil
}

// restoreAccount switches back to previous and describes the result; the
// returned value is always non-nil because it is reported either way.
func restoreAccount(gh ghops.GH, previous string) error {
	if err := gh.SwitchUser(githubHost, previous); err != nil {
		return fmt.Errorf("could not restore gh account %s: %w", previous, err)
	}
	return fmt.Errorf("restored gh account %s", previous)
}
