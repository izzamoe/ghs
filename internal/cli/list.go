package cli

import (
	"fmt"
	"text/tabwriter"
)

func (a App) listProfiles() error {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		_, err := fmt.Fprintln(a.out, "no profiles found")

		return err
	}

	w := tabwriter.NewWriter(a.out, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "PROFILE\tGH USER\tGIT EMAIL\tSSH ALIAS"); err != nil {
		return err
	}
	for _, profile := range cfg.Profiles {
		email := profile.GitEmail
		if email == "" {
			email = "(missing)"
		}
		if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", profile.Name, profile.GitHubUser, email, profile.SSHHostAlias); err != nil {
			return err
		}
	}

	return w.Flush()
}
