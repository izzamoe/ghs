package gitops

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// RemoteKind classifies a remote URL.
type RemoteKind int

const (
	// RemoteOther is anything that is not an SSH or HTTP(S) URL for a host
	// ghs can reason about, including non-URL strings.
	RemoteOther RemoteKind = iota
	// RemoteGitHub is github.com in scp, ssh://, https://, or http:// form.
	RemoteGitHub
	// RemoteSSHHost is scp or ssh:// form with any other host (possibly a
	// ghs profile alias).
	RemoteSSHHost
)

func (k RemoteKind) String() string {
	switch k {
	case RemoteGitHub:
		return "github"
	case RemoteSSHHost:
		return "ssh-host"
	default:
		return "other"
	}
}

// Remote is a parsed remote URL.
type Remote struct {
	Kind RemoteKind
	// Host is the lower-cased host name or scp host token.
	Host string
	// Path is owner/repo.git: no leading or trailing slash, exactly one .git.
	Path string
	Raw  string
}

// ParseRemote classifies a remote URL. Unparseable input is RemoteOther,
// not an error; only an empty URL is an error.
func ParseRemote(raw string) (Remote, error) {
	if strings.TrimSpace(raw) == "" {
		return Remote{}, errors.New("remote url is empty")
	}
	r := Remote{Kind: RemoteOther, Raw: raw}

	if scheme, rest, ok := strings.Cut(raw, "://"); ok {
		switch strings.ToLower(scheme) {
		case "ssh", "http", "https":
		default:
			return r, nil
		}
		u, err := url.Parse(strings.ToLower(scheme) + "://" + rest)
		if err != nil || u.Hostname() == "" {
			return r, nil
		}
		r.Host = strings.ToLower(u.Hostname())
		r.Path = normalizeRepoPath(u.Path)
		switch {
		case r.Path == "":
			r.Kind = RemoteOther
		case r.Host == "github.com":
			r.Kind = RemoteGitHub
		case strings.EqualFold(scheme, "ssh"):
			r.Kind = RemoteSSHHost
		}
		return r, nil
	}

	// scp-like syntax: [user@]host:path, where host contains no slash.
	hostPart, pathPart, ok := strings.Cut(raw, ":")
	if !ok || strings.Contains(hostPart, "/") || strings.ContainsAny(raw, " \t") {
		return r, nil
	}
	if at := strings.LastIndex(hostPart, "@"); at >= 0 {
		hostPart = hostPart[at+1:]
	}
	// A single letter before the colon is a Windows drive, not a host.
	if len(hostPart) < 2 {
		return r, nil
	}
	p := normalizeRepoPath(pathPart)
	if p == "" {
		return r, nil
	}
	r.Host = strings.ToLower(hostPart)
	r.Path = p
	if r.Host == "github.com" {
		r.Kind = RemoteGitHub
	} else {
		r.Kind = RemoteSSHHost
	}
	return r, nil
}

func normalizeRepoPath(p string) string {
	p = strings.Trim(p, "/")
	for strings.HasSuffix(p, ".git") {
		p = strings.TrimSuffix(p, ".git")
		p = strings.TrimRight(p, "/")
	}
	if p == "" {
		return ""
	}
	return p + ".git"
}

// AliasURL is the scp-style URL that routes this repository through alias.
func (r Remote) AliasURL(alias string) string {
	return "git@" + alias + ":" + r.Path
}

// RewriteGitHubURL rewrites a github.com URL, or a URL whose SSH host is one
// of profileAliases, to go through alias. Every other URL is rejected with
// an error naming its host.
func RewriteGitHubURL(raw string, alias string, profileAliases []string) (string, error) {
	if alias == "" {
		return "", errors.New("ssh host alias is required")
	}
	r, err := ParseRemote(raw)
	if err != nil {
		return "", err
	}
	switch r.Kind {
	case RemoteGitHub:
		return r.AliasURL(alias), nil
	case RemoteSSHHost:
		for _, a := range append([]string{alias}, profileAliases...) {
			if strings.EqualFold(r.Host, a) {
				return r.AliasURL(alias), nil
			}
		}
		return "", fmt.Errorf("remote %s uses ssh host %s, which is neither github.com nor a ghs profile alias", raw, r.Host)
	}
	if r.Host != "" {
		return "", fmt.Errorf("remote %s is not a GitHub remote (host %s)", raw, r.Host)
	}
	return "", fmt.Errorf("remote %s is not a GitHub remote", raw)
}

// isShorthand reports whether input is owner/repo.
func isShorthand(input string) bool {
	owner, repo, ok := strings.Cut(input, "/")
	return ok && owner != "" && repo != "" && !strings.ContainsAny(repo, "/:") &&
		!strings.ContainsAny(owner, ":@ \t") && !strings.ContainsAny(repo, " \t")
}

// CloneURL builds the alias URL for clone input: owner/repo shorthand or a
// github.com URL. Anything else is rejected, naming the host.
func CloneURL(input string, alias string) (string, error) {
	if alias == "" {
		return "", errors.New("ssh host alias is required")
	}
	if isShorthand(input) {
		return "git@" + alias + ":" + normalizeRepoPath(input), nil
	}
	r, err := ParseRemote(input)
	if err != nil {
		return "", err
	}
	if r.Kind != RemoteGitHub {
		if r.Host != "" {
			return "", fmt.Errorf("clone supports only github.com repositories; %s uses host %s", input, r.Host)
		}
		return "", fmt.Errorf("clone supports only owner/repo or a github.com url; got %q", input)
	}
	return r.AliasURL(alias), nil
}

// CloneDirectory infers the directory git clone would create.
func CloneDirectory(input string) string {
	var p string
	if isShorthand(input) {
		p = normalizeRepoPath(input)
	} else if r, err := ParseRemote(input); err == nil && r.Kind != RemoteOther {
		p = r.Path
	}
	if p == "" {
		return ""
	}
	return strings.TrimSuffix(path.Base(p), ".git")
}
