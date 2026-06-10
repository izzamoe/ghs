package ghops

import (
	"encoding/json"
	"fmt"

	"github.com/izzamoe/ghs/internal/runner"
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

type GH struct {
	runner runner.Runner
}

func New(run runner.Runner) GH {
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

	accounts := status.Hosts[hostname]
	if len(accounts) == 0 {
		return nil, fmt.Errorf("no gh accounts found for host %q", hostname)
	}

	return accounts, nil
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
