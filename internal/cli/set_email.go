package cli

import (
	"fmt"

	"github.com/izzamoe/ghs/internal/config"
)

func (a App) setEmail(pos []string) error {
	profileName := pos[0]
	email := pos[1]

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
