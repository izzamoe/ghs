package gitops

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/runner"
)

type Git struct {
	runner runner.Runner
}

func New(run runner.Runner) Git {
	return Git{runner: run}
}

func (g Git) SetIdentity(profile config.Profile, global bool) error {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	if err := g.runner.Run("git", append(args, "user.name", profile.GitName)...); err != nil {
		return err
	}
	return g.runner.Run("git", append(args, "user.email", profile.GitEmail)...)
}

func (g Git) SetIdentityInRepo(repoDir string, profile config.Profile) error {
	if err := g.runner.Run("git", "-C", repoDir, "config", "user.name", profile.GitName); err != nil {
		return err
	}
	return g.runner.Run("git", "-C", repoDir, "config", "user.email", profile.GitEmail)
}

func (g Git) Clone(url string, directory string) error {
	args := []string{"clone", url}
	if directory != "" {
		args = append(args, directory)
	}
	return g.runner.Run("git", args...)
}

func (g Git) OriginURL() (string, error) {
	return g.runner.Output("git", "remote", "get-url", "origin")
}

func (g Git) SetOriginURL(url string) error {
	return g.runner.Run("git", "remote", "set-url", "origin", url)
}

func (g Git) CurrentIdentity() (string, string, error) {
	name, err := g.runner.Output("git", "config", "user.name")
	if err != nil {
		return "", "", err
	}
	email, err := g.runner.Output("git", "config", "user.email")
	if err != nil {
		return "", "", err
	}
	return name, email, nil
}

func RewriteGitHubURL(url string, alias string) (string, error) {
	if alias == "" {
		return "", fmt.Errorf("ssh host alias is required")
	}

	if strings.HasPrefix(url, "git@github.com:") {
		return strings.Replace(url, "git@github.com:", "git@"+alias+":", 1), nil
	}
	if strings.HasPrefix(url, "ssh://git@github.com/") {
		return strings.Replace(url, "ssh://git@github.com/", "ssh://git@"+alias+"/", 1), nil
	}
	if path, ok := strings.CutPrefix(url, "https://github.com/"); ok {
		return "git@" + alias + ":" + path, nil
	}
	if path, ok := strings.CutPrefix(url, "http://github.com/"); ok {
		return "git@" + alias + ":" + path, nil
	}
	if strings.Contains(url, "@"+alias+":") || strings.Contains(url, "@"+alias+"/") {
		return url, nil
	}
	return "", fmt.Errorf("unsupported github remote url: %s", url)
}

func CloneURL(input string, alias string) (string, error) {
	if alias == "" {
		return "", fmt.Errorf("ssh host alias is required")
	}
	if strings.Count(input, "/") == 1 && !strings.Contains(input, ":") {
		return "git@" + alias + ":" + strings.TrimSuffix(input, ".git") + ".git", nil
	}
	return RewriteGitHubURL(input, alias)
}

func CloneDirectory(input string) string {
	cleaned := strings.TrimSuffix(input, ".git")
	if path, ok := strings.CutPrefix(cleaned, "https://github.com/"); ok {
		cleaned = path
	} else if path, ok := strings.CutPrefix(cleaned, "http://github.com/"); ok {
		cleaned = path
	} else if path, ok := strings.CutPrefix(cleaned, "ssh://git@github.com/"); ok {
		cleaned = path
	} else if path, ok := strings.CutPrefix(cleaned, "git@github.com:"); ok {
		cleaned = path
	} else if _, path, ok := strings.Cut(cleaned, ":"); ok && strings.HasPrefix(cleaned, "git@") {
		cleaned = path
	}
	base := filepath.Base(cleaned)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}
	return base
}
