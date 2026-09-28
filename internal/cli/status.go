package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

// status resolves the active account, the Git identity, and origin to
// profiles and prints one mismatch line per disagreement. It never mutates
// and always exits 0 once the config loads (FR-078).
func (a App) status() error {
	cfgPath, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}

	// gh account
	view := readAccounts()
	account := resolution{Kind: kindUnavailable}
	accountLine := ""
	switch {
	case !view.known:
		accountLine = fmt.Sprintf("unavailable (%s)", view.reason)
	case view.active == "":
		accountLine = "unavailable (no active account for github.com)"
	default:
		account = resolveLogin(cfg, view.active)
		accountLine = view.active + " " + describe(account)
	}

	// git identity and origin
	identity := resolution{Kind: kindUnavailable}
	var identityLine, originLine, email string
	origin := resolution{Kind: kindUnavailable}
	if err := requireTool("git"); err != nil {
		identityLine, originLine = "unavailable (git not found)", "unavailable (git not found)"
	} else {
		git := gitops.New(runner.New())
		_, inRepo, repoErr := git.InRepo()
		switch id, idErr := git.IdentityWithOrigin(!inRepo); {
		case repoErr != nil:
			identityLine = fmt.Sprintf("unavailable (%s)", shortReason(repoErr))
		case idErr != nil:
			identityLine = fmt.Sprintf("unavailable (%s)", shortReason(idErr))
		case id.Email == "":
			identityLine = "unavailable (user.email is not set)"
		default:
			email = id.Email
			identity = resolveEmail(cfg, id.Email)
			words := []string{scopeWord(id.EmailScope, inRepo)}
			if isIdentityFile(cfgPath, cfg, id.EmailFile) {
				words = append(words, "workspace")
			}
			identityLine = fmt.Sprintf("%s <%s> %s", id.Name, id.Email, describe(identity, words...))
		}

		switch url, exists, oErr := git.Origin(); {
		case repoErr != nil || !inRepo:
			originLine = "unavailable (not inside a git repository)"
		case oErr != nil:
			originLine = fmt.Sprintf("unavailable (%s)", shortReason(oErr))
		case !exists:
			originLine = "unavailable (no origin remote)"
		default:
			var remote gitops.Remote
			origin, remote = resolveOrigin(cfg, url)
			switch origin.Kind {
			case kindUnaliased:
				originLine = url + " (github.com, no alias)"
			case kindUnknownAlias:
				originLine = fmt.Sprintf("%s (unknown alias %s)", url, remote.Host)
			case kindNotGitHub:
				host := remote.Host
				if host == "" {
					host = "unknown host"
				}
				originLine = fmt.Sprintf("%s (not github: %s)", url, host)
			default:
				originLine = url + " " + describe(origin)
			}
		}
	}

	lines := []string{
		"ghs status",
		"----------",
		"gh account:    " + accountLine,
		"git identity:  " + identityLine,
		"origin:        " + originLine,
	}
	lines = append(lines, mismatches(view.active, email, account, identity, origin)...)
	for _, line := range lines {
		if err := a.printf("%s", line); err != nil {
			return err
		}
	}
	return nil
}

// describe renders a resolution as "(profile "p", extra...)" or
// "(no profile, extra...)".
func describe(r resolution, extra ...string) string {
	head := "no profile"
	if r.Kind == kindProfile {
		head = fmt.Sprintf("profile %q", r.Profile)
	}
	return "(" + strings.Join(append([]string{head}, extra...), ", ") + ")"
}

func scopeWord(scope string, inRepo bool) string {
	if scope != "" {
		return scope
	}
	if inRepo {
		return "local"
	}
	return "global"
}

// isIdentityFile reports whether file is one of the ghs identity files.
func isIdentityFile(cfgPath string, cfg config.Config, file string) bool {
	if file == "" {
		return false
	}
	for _, p := range cfg.Profiles {
		if strings.EqualFold(filepath.ToSlash(config.IdentityFilePath(cfgPath, p.Name)), filepath.ToSlash(file)) {
			return true
		}
	}
	return false
}

// mismatches lists the disagreements in the order fixed by contracts/cli.md.
func mismatches(login, email string, account, identity, origin resolution) []string {
	var out []string
	if account.Kind == kindNone {
		out = append(out, fmt.Sprintf("mismatch: gh account resolves to no profile (login %s)", login))
	}
	if identity.Kind == kindNone {
		out = append(out, fmt.Sprintf("mismatch: git identity resolves to no profile (%s)", email))
	}
	differ := func(a, b resolution) bool {
		return a.Kind == kindProfile && b.Kind == kindProfile && a.Profile != b.Profile
	}
	if differ(identity, account) {
		out = append(out, fmt.Sprintf("mismatch: git identity resolves to profile %q but gh account resolves to profile %q", identity.Profile, account.Profile))
	}
	if differ(origin, account) {
		out = append(out, fmt.Sprintf("mismatch: origin resolves to profile %q but gh account resolves to profile %q", origin.Profile, account.Profile))
	}
	if differ(origin, identity) {
		out = append(out, fmt.Sprintf("mismatch: origin resolves to profile %q but git identity resolves to profile %q", origin.Profile, identity.Profile))
	}
	if origin.Kind == kindUnaliased {
		target := "<profile>"
		if account.Kind == kindProfile {
			target = account.Profile
		}
		out = append(out, "mismatch: origin uses github.com directly; run: ghs fix-remote "+target)
	}
	return out
}
