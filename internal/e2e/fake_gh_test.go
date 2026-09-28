package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
)

// gh implements the "Fake gh" table of contracts/fake-tools.md.
func (f *fakeCtx) gh(args []string) int {
	gh := &f.st.GH
	switch {
	case len(args) == 6 && args[0] == "auth" && args[1] == "status" && args[2] == "--hostname" && args[4] == "--json" && args[5] == "hosts":
		if len(gh.Accounts) == 0 {
			return f.failf(1, "You are not logged into any GitHub hosts. To log in, run: gh auth login")
		}
		type jsonAccount struct {
			State  string `json:"state"`
			Active bool   `json:"active"`
			Host   string `json:"host"`
			Login  string `json:"login"`
		}
		hosts := map[string][]jsonAccount{}
		for _, a := range gh.Accounts {
			host := a.Host
			if host == "" {
				host = "github.com"
			}
			if host != args[3] {
				continue
			}
			hosts[host] = append(hosts[host], jsonAccount{State: a.State, Active: a.Active, Host: host, Login: a.Login})
		}
		data, _ := json.Marshal(map[string]any{"hosts": hosts})
		_, _ = fmt.Fprintln(f.stdout, string(data))
		return 0

	case len(args) == 6 && args[0] == "auth" && args[1] == "switch" && args[2] == "--hostname" && args[3] == "github.com" && args[4] == "--user":
		login := args[5]
		idx := slices.IndexFunc(gh.Accounts, func(a ghAccount) bool {
			return a.Login == login && (a.Host == "" || a.Host == "github.com")
		})
		if idx < 0 || gh.Accounts[idx].State != "success" {
			return f.failf(1, "could not switch: no healthy account %s on github.com", login)
		}
		for i := range gh.Accounts {
			if gh.Accounts[i].Host == "" || gh.Accounts[i].Host == "github.com" {
				gh.Accounts[i].Active = i == idx
			}
		}
		f.dirty = true
		_, _ = fmt.Fprintf(f.stdout, "✓ Switched active account for github.com to %s\n", login)
		return 0

	case len(args) == 2 && args[0] == "api" && args[1] == "user":
		login, ok := f.ghActive()
		if !ok {
			return f.failf(1, "gh: no active account")
		}
		u := gh.Users[login]
		if u.Login == "" {
			u.Login = login
		}
		data, _ := json.Marshal(map[string]any{"id": u.ID, "login": u.Login, "name": u.Name, "email": u.Email})
		_, _ = fmt.Fprintln(f.stdout, string(data))
		return 0

	case len(args) == 2 && args[0] == "api" && args[1] == "user/emails":
		login, ok := f.ghActive()
		if !ok {
			return f.failf(1, "gh: no active account")
		}
		emails := gh.Users[login].Emails
		if emails == nil {
			emails = []ghEmail{}
		}
		data, _ := json.Marshal(emails)
		_, _ = fmt.Fprintln(f.stdout, string(data))
		return 0

	case len(args) == 5 && args[0] == "ssh-key" && args[1] == "add" && args[3] == "--title":
		login, ok := f.ghActive()
		if !ok {
			return f.failf(1, "gh: no active account")
		}
		data, err := os.ReadFile(args[2])
		if err != nil {
			return f.failf(1, "failed to read public key: %v", err)
		}
		fields := strings.Fields(string(data))
		if len(fields) < 2 {
			return f.failf(1, "invalid public key")
		}
		key := fields[0] + " " + fields[1]
		for _, existing := range gh.SSHKeys[login] {
			ef := strings.Fields(existing)
			if len(ef) >= 2 && ef[0]+" "+ef[1] == key {
				_, _ = fmt.Fprintln(f.stdout, "✓ Public key already exists on your account")
				return 0
			}
		}
		if gh.SSHKeys == nil {
			gh.SSHKeys = map[string][]string{}
		}
		gh.SSHKeys[login] = append(gh.SSHKeys[login], strings.TrimSpace(string(data)))
		f.dirty = true
		_, _ = fmt.Fprintln(f.stdout, "✓ Public key added to your account")
		return 0

	case len(args) >= 2 && args[0] == "auth" && args[1] == "logout":
		return f.failf(1, "fake gh: logout must never be called")
	}
	return f.unsupported(args)
}

func (f *fakeCtx) ghActive() (string, bool) {
	for _, a := range f.st.GH.Accounts {
		if a.Active && (a.Host == "" || a.Host == "github.com") {
			return a.Login, true
		}
	}
	return "", false
}
