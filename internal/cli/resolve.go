package cli

import (
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
