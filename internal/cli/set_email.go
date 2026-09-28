package cli

import (
	"github.com/izzamoe/ghs/internal/config"
)

func (a App) setEmail(pos []string) error {
	profileName, email := pos[0], pos[1]
	if err := config.ValidateEmail(email); err != nil {
		return usageErrorf("set-email", "%v", err)
	}
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	_, idx, err := findProfile(cfg, profileName)
	if err != nil {
		return err
	}
	cfg.Profiles[idx].GitEmail = email
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	return a.printf("set email for profile %q", profileName)
}
