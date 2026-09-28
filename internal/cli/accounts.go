package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/runner"
)

// githubHost is the only supported host (constitution Principle IV).
const githubHost = "github.com"

var errGHMissing = errors.New("gh is not installed; see https://cli.github.com")

// requireTool fails with an actionable message when name is not on PATH.
func requireTool(name string) error {
	if _, err := runner.New().LookPath(name); err != nil {
		if name == "gh" {
			return errGHMissing
		}
		return fmt.Errorf("%s is not installed or not on PATH", name)
	}
	return nil
}

// accountPreflight confirms, before any mutation, that the profile's
// account is logged in and healthy for github.com (FR-020). It returns the
// currently active login ("" when none).
func accountPreflight(gh ghops.GH, p config.Profile) (active string, err error) {
	if p.GitHubUser == "" {
		return "", fmt.Errorf("profile %q has no gh_user; add it with: ghs add-profile %s --gh-user <login> ...", p.Name, p.Name)
	}
	if err := config.ValidateLogin(p.GitHubUser); err != nil {
		return "", fmt.Errorf("profile %q: %w", p.Name, err)
	}
	if err := requireTool("gh"); err != nil {
		return "", err
	}
	accounts, err := gh.AuthAccounts(githubHost)
	if err != nil {
		return "", fmt.Errorf("cannot read gh auth status: %w", err)
	}
	if ghops.FindAccount(accounts, p.GitHubUser).State != "success" {
		return "", fmt.Errorf("gh account %s is not logged in for github.com; run: gh auth login --hostname github.com", p.GitHubUser)
	}
	active, _, _ = ghops.ActiveLogin(accounts)
	return active, nil
}

func equalLogin(a, b string) bool {
	return a != "" && b != "" && strings.EqualFold(a, b)
}

func displayLogin(login string) string {
	if login == "" {
		return "(none)"
	}
	return login
}
