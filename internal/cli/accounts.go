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

// accountView is a read-only snapshot of the GitHub CLI accounts for
// github.com, degraded to "unknown" when gh is missing or fails (FR-079).
type accountView struct {
	accounts []ghops.AuthAccount
	known    bool
	reason   string // why the view is unknown
	active   string // "" when no account is active
}

func readAccounts() accountView {
	if err := requireTool("gh"); err != nil {
		return accountView{reason: "gh not found"}
	}
	accounts, err := ghops.New(runner.New()).AuthAccounts(githubHost)
	if err != nil {
		return accountView{reason: shortReason(err)}
	}
	active, _, _ := ghops.ActiveLogin(accounts)
	return accountView{accounts: accounts, known: true, active: active}
}

// shortReason reduces "run gh: exit status 1: <message>" to the tool's own
// message, first line only.
func shortReason(err error) string {
	s := err.Error()
	if _, rest, ok := strings.Cut(s, ": exit status "); ok {
		if _, msg, ok := strings.Cut(rest, ": "); ok {
			s = msg
		}
	}
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}
