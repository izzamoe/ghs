package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/izzamoe/ghs/internal/ghops"
)

// listProfiles prints the profile table with the active-profile marker and
// each profile's GitHub CLI state, then the active account line. It never
// mutates and works without the GitHub CLI (FR-077, FR-079).
func (a App) listProfiles() error {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	view := readAccounts()

	if len(cfg.Profiles) == 0 {
		if err := a.printf("no profiles found"); err != nil {
			return err
		}
	} else {
		w := tabwriter.NewWriter(a.out, 0, 0, 2, ' ', 0)
		if _, err := fmt.Fprintln(w, "\tPROFILE\tGH USER\tGIT EMAIL\tSSH ALIAS\tWORKSPACE\tGH AUTH"); err != nil {
			return err
		}
		for _, p := range cfg.Profiles {
			// A space keeps the marker column one character wide when no row
			// is active, so the table layout never shifts.
			marker, auth := " ", "?"
			if view.known {
				account := ghops.FindAccount(view.accounts, p.GitHubUser)
				switch {
				case account.Active && account.State == "success":
					marker, auth = "*", "active"
				case account.State == "success":
					auth = "yes"
				default:
					auth = "no"
				}
			}
			email := p.GitEmail
			if email == "" {
				email = "(missing)"
			}
			workspace := p.Workspace
			if workspace == "" {
				workspace = "-"
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", marker, p.Name, p.GitHubUser, email, p.SSHHostAlias, workspace, auth); err != nil {
				return err
			}
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}

	switch {
	case !view.known:
		return a.printf("active gh account: unknown (%s)", view.reason)
	case view.active == "":
		return a.printf("active gh account: none for github.com")
	}
	if p, ok := cfg.ByLogin(view.active); ok {
		return a.printf("active gh account: %s (profile %q)", view.active, p.Name)
	}
	return a.printf("active gh account: %s (no profile matches)", view.active)
}
