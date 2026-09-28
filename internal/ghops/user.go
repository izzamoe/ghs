package ghops

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type User struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type AuthAccount struct {
	State  string `json:"state"`
	Active bool   `json:"active"`
	Host   string `json:"host"`
	Login  string `json:"login"`
}

type Email struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility"`
}

// commandRunner is the subset of runner.Runner that GH needs; unit tests
// substitute a stub.
type commandRunner interface {
	Run(name string, args ...string) error
	OutputBytes(name string, args ...string) ([]byte, error)
	CombinedOutput(name string, args ...string) (string, int, error)
}

type GH struct {
	runner commandRunner
}

func New(run commandRunner) GH {
	return GH{runner: run}
}

func (g GH) ActiveUser() (User, error) {
	data, err := g.runner.OutputBytes("gh", "api", "user")
	if err != nil {
		return User{}, err
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return User{}, fmt.Errorf("parse gh user: %w", err)
	}
	if user.Login == "" {
		return User{}, fmt.Errorf("gh api user did not return login")
	}
	if user.Email == "" {
		if email, err := g.PrimaryEmail(); err == nil {
			user.Email = email
		}
	}

	return user, nil
}

func (g GH) AuthAccounts(hostname string) ([]AuthAccount, error) {
	data, err := g.runner.OutputBytes("gh", "auth", "status", "--hostname", hostname, "--json", "hosts")
	if err != nil {
		return nil, err
	}

	return ParseAuthAccounts(hostname, data)
}

func ParseAuthAccounts(hostname string, data []byte) ([]AuthAccount, error) {
	var status struct {
		Hosts map[string][]AuthAccount `json:"hosts"`
	}
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("parse gh auth status: %w", err)
	}

	return status.Hosts[hostname], nil
}

// ActiveAccount returns the login of the active account for hostname; ok is
// false when the GitHub CLI reports no active account there.
func (g GH) ActiveAccount(hostname string) (login string, ok bool, err error) {
	accounts, err := g.AuthAccounts(hostname)
	if err != nil {
		return "", false, err
	}
	return ActiveLogin(accounts)
}

// ActiveLogin picks the active account's login from an account list.
func ActiveLogin(accounts []AuthAccount) (string, bool, error) {
	for _, account := range accounts {
		if account.Active && account.Login != "" {
			return account.Login, true, nil
		}
	}
	return "", false, nil
}

// IsAuthenticated reports whether login is logged in and healthy (state
// "success") for hostname.
func (g GH) IsAuthenticated(hostname string, login string) (bool, error) {
	accounts, err := g.AuthAccounts(hostname)
	if err != nil {
		return false, err
	}
	return FindAccount(accounts, login).State == "success", nil
}

// FindAccount returns the account with this login (case-insensitive), or the
// zero value.
func FindAccount(accounts []AuthAccount, login string) AuthAccount {
	for _, account := range accounts {
		if login != "" && strings.EqualFold(account.Login, login) {
			return account
		}
	}
	return AuthAccount{}
}

// WithAccount runs fn while target is the active account and then restores
// the previously active account, on success and on failure. No switch is
// made when target is already active. switched reports whether a switch
// happened. The returned error names both fn's failure and the result of
// the restore.
func (g GH) WithAccount(hostname string, target string, fn func() error) (switched bool, err error) {
	previous, _, err := g.ActiveAccount(hostname)
	if err != nil {
		return false, err
	}
	if strings.EqualFold(previous, target) {
		return false, fn()
	}
	if err := g.SwitchUser(hostname, target); err != nil {
		return false, fmt.Errorf("switch gh account to %s: %w", target, err)
	}
	fnErr := fn()
	if previous == "" {
		return true, fnErr
	}
	return true, RestoreResult(fnErr, g.SwitchUser(hostname, previous), previous)
}

// RestoreResult combines an operation's error with the outcome of switching
// back to previous, so both are always reported.
func RestoreResult(opErr error, restoreErr error, previous string) error {
	switch {
	case opErr == nil && restoreErr == nil:
		return nil
	case restoreErr != nil:
		return errors.Join(opErr, fmt.Errorf("could not restore gh account %s: %w", previous, restoreErr))
	default:
		return errors.Join(opErr, fmt.Errorf("restored gh account %s", previous))
	}
}

// AddSSHKey uploads a public key to the active account. A key that is
// already registered is reported through alreadyPresent, not as an error.
func (g GH) AddSSHKey(pubPath string, title string) (alreadyPresent bool, err error) {
	out, code, err := g.runner.CombinedOutput("gh", "ssh-key", "add", pubPath, "--title", title)
	if err != nil {
		return false, err
	}
	already := strings.Contains(strings.ToLower(out), "already")
	if code == 0 {
		return already, nil
	}
	if already {
		return true, nil
	}
	return false, fmt.Errorf("gh ssh-key add failed (exit %d): %s", code, strings.TrimSpace(out))
}

func (g GH) SwitchUser(hostname string, login string) error {
	return g.runner.Run("gh", "auth", "switch", "--hostname", hostname, "--user", login)
}

func (g GH) PrimaryEmail() (string, error) {
	data, err := g.runner.OutputBytes("gh", "api", "user/emails")
	if err != nil {
		return "", err
	}

	var emails []Email
	if err := json.Unmarshal(data, &emails); err != nil {
		return "", fmt.Errorf("parse gh emails: %w", err)
	}

	return SelectPrimaryEmail(emails), nil
}

func SelectPrimaryEmail(emails []Email) string {
	for _, email := range emails {
		if email.Primary && email.Verified && email.Email != "" {
			return email.Email
		}
	}
	for _, email := range emails {
		if email.Verified && email.Email != "" {
			return email.Email
		}
	}
	for _, email := range emails {
		if email.Email != "" {
			return email.Email
		}
	}

	return ""
}

func NoreplyEmail(user User) string {
	if user.ID <= 0 || user.Login == "" {
		return ""
	}

	return fmt.Sprintf("%d+%s@users.noreply.github.com", user.ID, user.Login)
}
