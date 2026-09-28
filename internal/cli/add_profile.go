package cli

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/runner"
)

var unsafeProfilePathChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (a App) addProfile(pos []string, flags map[string]string) error {
	profile := config.Profile{Name: pos[0]}
	profile.GitHubUser = flags["gh-user"]
	profile.GitName = flags["git-name"]
	profile.GitEmail = flags["git-email"]
	profile.SSHHostAlias = flags["ssh-alias"]
	profile.SSHKey = flags["ssh-key"]
	profile.Workspace = flags["workspace"]
	if err := validateCompleteProfile(profile); err != nil {
		return err
	}
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	return a.saveProfile(path, cfg, profile)
}

func (a App) addFromGH(pos []string, flags map[string]string) error {
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	user, err := ghops.New(runner.New()).ActiveUser()
	if err != nil {
		return err
	}
	profile, err := profileFromGH(pos[0], user, flags)
	if err != nil {
		return err
	}
	if err := validateImportedProfile(profile); err != nil {
		return err
	}
	return a.saveProfile(path, cfg, profile)
}

func (a App) importAll(flags map[string]string) error {
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	hostname := cmp.Or(flags["hostname"], "github.com")
	gh := ghops.New(runner.New())
	accounts, err := gh.AuthAccounts(hostname)
	if err != nil {
		return err
	}

	activeLogin := activeAccountLogin(accounts)
	profiles := make([]config.Profile, 0, len(accounts))
	for _, account := range accounts {
		if account.State != "success" || account.Login == "" {
			continue
		}
		if err := gh.SwitchUser(hostname, account.Login); err != nil {
			return restoreActive(gh, hostname, activeLogin, err)
		}
		user, err := gh.ActiveUser()
		if err != nil {
			return restoreActive(gh, hostname, activeLogin, err)
		}
		profile, err := profileFromGH(user.Login, user, flags)
		if err != nil {
			return restoreActive(gh, hostname, activeLogin, err)
		}
		if err := validateImportedProfile(profile); err != nil {
			return restoreActive(gh, hostname, activeLogin, err)
		}
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return restoreActive(gh, hostname, activeLogin, fmt.Errorf("no healthy gh accounts found for host %q", hostname))
	}
	mergeProfiles(&cfg, profiles, hasFlagKey(flags, "no-overwrite"))
	if err := config.Save(path, cfg); err != nil {
		return restoreActive(gh, hostname, activeLogin, err)
	}

	restoreErr := restoreActive(gh, hostname, activeLogin, nil)
	if restoreErr != nil {
		return restoreErr
	}
	_, err = fmt.Fprintf(a.out, "imported %d profiles from gh host %q\n", len(profiles), hostname)

	return err
}

// saveProfile merges profile into the loaded cfg and saves it atomically.
func (a App) saveProfile(path string, cfg config.Config, profile config.Profile) error {
	mergeProfiles(&cfg, []config.Profile{profile}, false)
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	return a.printf("saved profile %q to %s", profile.Name, path)
}

// mergeProfiles adds new profiles at the end and replaces same-named ones
// (unless noOverwrite). A replaced profile keeps its unknown keys and, when
// the replacement names none, its workspace (FR-005, FR-060).
func mergeProfiles(cfg *config.Config, profiles []config.Profile, noOverwrite bool) {
	for _, profile := range profiles {
		idx := cfg.Index(profile.Name)
		if idx < 0 {
			cfg.Profiles = append(cfg.Profiles, profile)
			continue
		}
		if noOverwrite {
			continue
		}
		old := cfg.Profiles[idx]
		if profile.Workspace == "" {
			profile.Workspace = old.Workspace
		}
		if len(profile.Extra) == 0 {
			profile.Extra = old.Extra
		}
		cfg.Profiles[idx] = profile
	}
}

func profileFromGH(name string, user ghops.User, flags map[string]string) (config.Profile, error) {
	if name == "" {
		return config.Profile{}, fmt.Errorf("profile name is required")
	}
	gitEmail := cmp.Or(flags["git-email"], user.Email, ghops.NoreplyEmail(user))
	if gitEmail == "" && hasFlagKey(flags, "require-email") {
		return config.Profile{}, fmt.Errorf("git email is required because gh did not return an email; run `gh auth refresh --scopes user:email` or pass --git-email")
	}

	safeName := safeProfilePathName(name)
	return config.Profile{
		Name:         name,
		GitHubUser:   user.Login,
		GitName:      cmp.Or(flags["git-name"], user.Name, user.Login),
		GitEmail:     gitEmail,
		SSHHostAlias: cmp.Or(flags["ssh-alias"], "github-"+safeName),
		SSHKey:       cmp.Or(flags["ssh-key"], "~/.ssh/id_ed25519_"+safeName),
		Workspace:    flags["workspace"],
	}, nil
}

func activeAccountLogin(accounts []ghops.AuthAccount) string {
	for _, account := range accounts {
		if account.Active {
			return account.Login
		}
	}

	return ""
}

func restoreActive(gh ghops.GH, hostname string, activeLogin string, err error) error {
	if activeLogin == "" {
		return err
	}
	restoreErr := gh.SwitchUser(hostname, activeLogin)
	if err != nil {
		return errors.Join(err, restoreErr)
	}

	return restoreErr
}

func safeProfilePathName(name string) string {
	cleaned := strings.Trim(unsafeProfilePathChars.ReplaceAllString(name, "-"), "-")
	if cleaned == "" {
		return "profile"
	}
	return strings.ToLower(cleaned)
}

func validateImportedProfile(profile config.Profile) error {
	if profile.Name == "" || profile.GitHubUser == "" || profile.GitName == "" || profile.SSHHostAlias == "" || profile.SSHKey == "" {
		return fmt.Errorf("profile requires name, gh-user, git-name, ssh-alias, and ssh-key")
	}
	return nil
}

func validateCompleteProfile(profile config.Profile) error {
	if err := validateImportedProfile(profile); err != nil {
		return err
	}
	if profile.GitEmail == "" {
		return fmt.Errorf("profile requires git-email")
	}
	return nil
}

func hasFlagKey(flags map[string]string, key string) bool {
	_, ok := flags[key]
	return ok
}
