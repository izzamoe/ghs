package gitops

import (
	"errors"
	"fmt"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
)

// commandRunner is the subset of runner.Runner that Git needs; unit tests
// substitute a stub whose errors implement ExitCode() int.
type commandRunner interface {
	Run(name string, args ...string) error
	Output(name string, args ...string) (string, error)
}

type Git struct {
	runner commandRunner
}

func New(run commandRunner) Git {
	return Git{runner: run}
}

// exitCode returns the exit status carried by err, -1 when there is none.
func exitCode(err error) int {
	var coded interface{ ExitCode() int }
	if errors.As(err, &coded) {
		return coded.ExitCode()
	}
	return -1
}

// InRepo reports whether the current directory is inside a Git work tree and
// returns its top-level directory. Not being in a repository is not an
// error; a missing or failing git is.
func (g Git) InRepo() (toplevel string, ok bool, err error) {
	out, err := g.runner.Output("git", "rev-parse", "--show-toplevel")
	if err != nil {
		if exitCode(err) == 128 {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

// Identity is the effective user.name and user.email with where each came
// from, as reported by git config --show-origin --show-scope.
type Identity struct {
	Name       string
	Email      string
	NameFile   string
	EmailFile  string
	NameScope  string
	EmailScope string
}

// IdentityWithOrigin reads user.name and user.email for the current
// directory, or from the global scope only when global is true. An unset
// key leaves its fields empty.
func (g Git) IdentityWithOrigin(global bool) (Identity, error) {
	var id Identity
	for _, key := range []string{"user.name", "user.email"} {
		args := []string{"config"}
		if global {
			args = append(args, "--global")
		}
		args = append(args, "--show-origin", "--show-scope", "--get", key)
		out, err := g.runner.Output("git", args...)
		if err != nil {
			if exitCode(err) == 1 {
				continue
			}
			return Identity{}, err
		}
		scope, file, value, ok := ParseShowOriginLine(out)
		if !ok {
			return Identity{}, fmt.Errorf("unexpected git config output: %q", out)
		}
		if key == "user.name" {
			id.Name, id.NameFile, id.NameScope = value, file, scope
		} else {
			id.Email, id.EmailFile, id.EmailScope = value, file, scope
		}
	}
	return id, nil
}

// ParseShowOriginLine splits "<scope>\t<origin>\t<value>"; a "file:" origin
// is reduced to the path.
func ParseShowOriginLine(line string) (scope, file, value string, ok bool) {
	line = strings.TrimRight(line, "\r\n")
	parts := strings.SplitN(line, "\t", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], strings.TrimPrefix(parts[1], "file:"), parts[2], true
}

// SetIdentity writes user.name then user.email locally (or globally). On
// failure step names the key that could not be written.
func (g Git) SetIdentity(profile config.Profile, global bool) (step string, err error) {
	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	if err := g.runner.Run("git", append(args, "user.name", profile.GitName)...); err != nil {
		return "user.name", err
	}
	if err := g.runner.Run("git", append(args, "user.email", profile.GitEmail)...); err != nil {
		return "user.email", err
	}
	return "", nil
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

// Origin returns the origin URL; exists is false when there is no origin
// remote.
func (g Git) Origin() (url string, exists bool, err error) {
	out, err := g.runner.Output("git", "remote", "get-url", "origin")
	if err != nil {
		if exitCode(err) == 2 {
			return "", false, nil
		}
		return "", false, err
	}
	return strings.TrimSpace(out), true, nil
}

func (g Git) SetOriginURL(url string) error {
	return g.runner.Run("git", "remote", "set-url", "origin", url)
}
