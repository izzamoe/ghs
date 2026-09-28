package cli

import (
	"errors"
	"fmt"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/gitops"
)

// resolution is what one fact (active account, Git identity, origin)
// resolves to. list, status, doctor, use, and fix-remote share these rules.
type resolution struct {
	Profile string
	Kind    string
}

const (
	kindProfile      = "profile"      // resolves to Profile
	kindNone         = "none"         // present but matches no profile
	kindUnavailable  = "unavailable"  // the fact could not be read
	kindUnaliased    = "unaliased"    // origin uses github.com directly
	kindNotGitHub    = "notgithub"    // origin is not a GitHub remote
	kindUnknownAlias = "unknownalias" // origin uses an SSH host no profile owns
)

func resolveLogin(cfg config.Config, login string) resolution {
	if login == "" {
		return resolution{Kind: kindUnavailable}
	}
	if p, ok := cfg.ByLogin(login); ok {
		return resolution{Profile: p.Name, Kind: kindProfile}
	}
	return resolution{Kind: kindNone}
}

func resolveEmail(cfg config.Config, email string) resolution {
	if email == "" {
		return resolution{Kind: kindUnavailable}
	}
	if p, ok := cfg.ByEmail(email); ok {
		return resolution{Profile: p.Name, Kind: kindProfile}
	}
	return resolution{Kind: kindNone}
}

func resolveAlias(cfg config.Config, alias string) resolution {
	if p, ok := cfg.ByAlias(alias); ok {
		return resolution{Profile: p.Name, Kind: kindProfile}
	}
	return resolution{Kind: kindUnknownAlias}
}

// resolveOrigin classifies an origin URL against the config. An empty URL
// (no origin) resolves to kindNone.
func resolveOrigin(cfg config.Config, url string) (resolution, gitops.Remote) {
	if url == "" {
		return resolution{Kind: kindNone}, gitops.Remote{}
	}
	remote, err := gitops.ParseRemote(url)
	if err != nil {
		return resolution{Kind: kindNotGitHub}, remote
	}
	switch remote.Kind {
	case gitops.RemoteGitHub:
		return resolution{Kind: kindUnaliased}, remote
	case gitops.RemoteSSHHost:
		return resolveAlias(cfg, remote.Host), remote
	}
	return resolution{Kind: kindNotGitHub}, remote
}

// originRewrite decides what fix-remote does with url for profile p: a
// github.com URL or another profile's alias is rewritten to p's alias; p's
// own alias is already correct; anything else is refused, naming the host.
func originRewrite(cfg config.Config, p config.Profile, url string) (newURL string, alreadyCorrect bool, err error) {
	res, remote := resolveOrigin(cfg, url)
	switch res.Kind {
	case kindNone:
		return "", false, errors.New("no origin remote; nothing to rewrite")
	case kindUnaliased:
		return remote.AliasURL(p.SSHHostAlias), false, nil
	case kindProfile:
		if res.Profile == p.Name {
			return url, true, nil
		}
		return remote.AliasURL(p.SSHHostAlias), false, nil
	case kindUnknownAlias:
		return "", false, fmt.Errorf("origin %s uses ssh host %s, which is neither github.com nor the alias of a ghs profile; not rewriting it", url, remote.Host)
	}
	host := remote.Host
	if host == "" {
		host = "unknown"
	}
	return "", false, fmt.Errorf("origin %s is not a GitHub remote (host %s); only github.com remotes are rewritten", url, host)
}

// originWarning is the stderr warning ghs use prints when origin bypasses
// the profile's alias, or "" when there is nothing to warn about (FR-049).
func originWarning(cfg config.Config, p config.Profile, url string) string {
	res, _ := resolveOrigin(cfg, url)
	remedy := fmt.Sprintf("run: ghs use %s --fix-remote  or  ghs fix-remote %s", p.Name, p.Name)
	switch {
	case res.Kind == kindUnaliased:
		return fmt.Sprintf("origin %s still uses github.com; pushes will not use the key of profile %q; %s", url, p.Name, remedy)
	case res.Kind == kindProfile && res.Profile != p.Name:
		return fmt.Sprintf("origin %s uses the alias of profile %q; %s", url, res.Profile, remedy)
	}
	return ""
}
