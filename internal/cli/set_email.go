package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/config"
)

func (a App) setEmail(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: ghs set-email <profile> <email>")
	}
	profileName := args[0]
	email := args[1]

	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	updated := false
	for i, profile := range cfg.Profiles {
		if profile.Name == profileName {
			cfg.Profiles[i].GitEmail = email
			updated = true
			break
		}
	}
	if !updated {
		return fmt.Errorf("profile %q not found", profileName)
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "set email for profile %q\n", profileName)

	return err
}
