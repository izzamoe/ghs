package cli

import (
	"cmp"
	"errors"
	"fmt"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/runner"
)

func (a App) addProfile(pos []string, flags map[string]string) error {
	profile := config.Profile{
		Name:         pos[0],
		GitHubUser:   flags["gh-user"],
		GitName:      flags["git-name"],
		GitEmail:     flags["git-email"],
		SSHHostAlias: flags["ssh-alias"],
		SSHKey:       flags["ssh-key"],
	}
	for _, required := range []string{"gh-user", "git-name", "git-email", "ssh-alias", "ssh-key"} {
		if _, ok := flags[required]; !ok {
			return usageErrorf("add-profile", "missing required flag --%s", required)
		}
	}
	if err := validateFlagValues("add-profile", profile.Name, flags); err != nil {
		return err
	}

	profile.Workspace = config.NormalizeWorkspace(flags["workspace"])

	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	if err := checkProfileChange(cfg, profile); err != nil {
		return err
	}
	return a.saveProfile(path, cfg, profile)
}

func (a App) addFromGH(pos []string, flags map[string]string) error {
	name := pos[0]
	if err := validateFlagValues("add-from-gh", name, flags); err != nil {
		return err
	}
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	user, err := ghops.New(runner.New()).ActiveUser()
	if err != nil {
		return err
	}
	profile, err := profileFromGH(name, user, flags)
	if err != nil {
		return err
	}
	if err := validateGHProfile(profile, flags); err != nil {
		return err
	}
	profile.Workspace = config.NormalizeWorkspace(profile.Workspace)
	if profile.Workspace != "" && profile.GitEmail == "" {
		return errors.New("--workspace needs a git email and gh returned none; pass --git-email")
	}
	if err := checkProfileChange(cfg, profile); err != nil {
		return err
	}
	return a.saveProfile(path, cfg, profile)
}

// checkProfileChange rejects collisions with other profiles and a changed
// workspace on an already-linked profile (which would orphan its link).
func checkProfileChange(cfg config.Config, profile config.Profile) error {
	idx := cfg.Index(profile.Name)
	if err := cfg.CheckUnique(profile, idx); err != nil {
		return err
	}
	if idx >= 0 && profile.Workspace != "" {
		if old := cfg.Profiles[idx].Workspace; old != "" && !strings.EqualFold(config.NormalizeWorkspace(old), profile.Workspace) {
			return fmt.Errorf("profile %q is already linked to %s; run: ghs workspace %s --unlink first", profile.Name, old, profile.Name)
		}
	}
	return nil
}

func (a App) importAll(flags map[string]string) error {
	// Only github.com is supported end to end; any other host is refused
	// before the GitHub CLI is called (FR-033).
	if h, ok := flags["hostname"]; ok && !strings.EqualFold(h, githubHost) {
		return usageErrorf("import-all", "only github.com is supported; got --hostname %q", h)
	}
	hostname := githubHost
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	gh := ghops.New(runner.New())
	accounts, err := gh.AuthAccounts(hostname)
	if err != nil {
		return err
	}

	activeLogin := activeAccountLogin(accounts)
	merged := cfg
	merged.Profiles = append([]config.Profile(nil), cfg.Profiles...)
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
		if err := config.ValidateName(profile.Name, reservedNames()); err != nil {
			return restoreActive(gh, hostname, activeLogin, fmt.Errorf("account %s: %w", account.Login, err))
		}
		if err := validateGHProfile(profile, flags); err != nil {
			return restoreActive(gh, hostname, activeLogin, fmt.Errorf("account %s: %w", account.Login, err))
		}
		idx := merged.Index(profile.Name)
		if idx >= 0 && hasFlagKey(flags, "no-overwrite") {
			continue
		}
		if err := merged.CheckUnique(profile, idx); err != nil {
			return restoreActive(gh, hostname, activeLogin, fmt.Errorf("account %s: %w", account.Login, err))
		}
		mergeProfiles(&merged, []config.Profile{profile}, false)
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 && !hasFlagKey(flags, "no-overwrite") {
		return restoreActive(gh, hostname, activeLogin, fmt.Errorf("no healthy gh accounts found for host %q", hostname))
	}
	if err := config.Save(path, merged); err != nil {
		return restoreActive(gh, hostname, activeLogin, err)
	}

	restoreErr := restoreActive(gh, hostname, activeLogin, nil)
	if restoreErr != nil {
		restoreErr = fmt.Errorf("imported %d profiles, but %w", len(profiles), restoreErr)
	} else if err := a.printf("imported %d profiles from gh host %q", len(profiles), hostname); err != nil {
		return err
	}
	for _, p := range profiles {
		line, err := refreshIdentityFile(path, merged.Profiles[merged.Index(p.Name)])
		if err != nil {
			return errors.Join(restoreErr, err)
		}
		if line != "" {
			if err := a.printf("%s", line); err != nil {
				return err
			}
		}
	}
	return restoreErr
}

// validateFlagValues validates the profile name (when non-empty) and every
// value flag given on the command line; failures are usage errors.
func validateFlagValues(command string, name string, flags map[string]string) error {
	if name != "" {
		if err := config.ValidateName(name, reservedNames()); err != nil {
			return usageErrorf(command, "%v", err)
		}
	}
	validators := map[string]func(string) error{
		"gh-user":   config.ValidateLogin,
		"git-name":  config.ValidateGitName,
		"git-email": config.ValidateEmail,
		"ssh-alias": config.ValidateAlias,
		"ssh-key":   config.ValidateKeyPath,
		"workspace": config.ValidateWorkspace,
	}
	for _, flag := range []string{"gh-user", "git-name", "git-email", "ssh-alias", "ssh-key", "workspace"} {
		value, ok := flags[flag]
		if !ok {
			continue
		}
		if err := validators[flag](value); err != nil {
			return usageErrorf(command, "--%s: %v", flag, err)
		}
	}
	return nil
}

// validateGHProfile validates the values that came from the GitHub CLI or
// were derived from it (flag values were validated already). These are not
// usage errors: the user did not type them.
func validateGHProfile(p config.Profile, flags map[string]string) error {
	if err := config.ValidateLogin(p.GitHubUser); err != nil {
		return fmt.Errorf("gh returned an unusable login: %w", err)
	}
	if _, ok := flags["git-name"]; !ok {
		if err := config.ValidateGitName(p.GitName); err != nil {
			return fmt.Errorf("gh returned an unusable git name: %w; pass --git-name", err)
		}
	}
	if _, ok := flags["git-email"]; !ok && p.GitEmail != "" {
		if err := config.ValidateEmail(p.GitEmail); err != nil {
			return fmt.Errorf("gh returned an unusable email: %w; pass --git-email", err)
		}
	}
	if _, ok := flags["ssh-alias"]; !ok {
		if err := config.ValidateAlias(p.SSHHostAlias); err != nil {
			return fmt.Errorf("derived %w; pass --ssh-alias", err)
		}
	}
	if _, ok := flags["ssh-key"]; !ok {
		if err := config.ValidateKeyPath(p.SSHKey); err != nil {
			return fmt.Errorf("derived %w; pass --ssh-key", err)
		}
	}
	return nil
}

// saveProfile merges profile into the loaded cfg and saves it atomically.
// When the profile names a workspace it is then linked; a link failure is
// reported with the command that completes it, because the saved profile
// is correct and linking is idempotent. An existing identity file of a
// linked profile is refreshed (FR-060).
func (a App) saveProfile(path string, cfg config.Config, profile config.Profile) error {
	mergeProfiles(&cfg, []config.Profile{profile}, false)
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	if err := a.printf("saved profile %q to %s", profile.Name, path); err != nil {
		return err
	}
	idx := cfg.Index(profile.Name)
	if profile.Workspace != "" {
		if err := a.workspaceLink(path, cfg, idx, profile.Workspace); err != nil {
			return fmt.Errorf("workspace link failed: %w; rerun: ghs workspace %s %s", err, profile.Name, profile.Workspace)
		}
		return nil
	}
	line, err := refreshIdentityFile(path, cfg.Profiles[idx])
	if err != nil || line == "" {
		return err
	}
	return a.printf("%s", line)
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

	suffix := config.SafeSuffix(name)
	return config.Profile{
		Name:         name,
		GitHubUser:   user.Login,
		GitName:      cmp.Or(flags["git-name"], user.Name, user.Login),
		GitEmail:     gitEmail,
		SSHHostAlias: cmp.Or(flags["ssh-alias"], "github-"+suffix),
		SSHKey:       cmp.Or(flags["ssh-key"], "~/.ssh/id_ed25519_"+suffix),
		Workspace:    flags["workspace"],
	}, nil
}

func activeAccountLogin(accounts []ghops.AuthAccount) string {
	login, _, _ := ghops.ActiveLogin(accounts)
	return login
}

// restoreActive switches back to activeLogin and reports both err and the
// restore result.
func restoreActive(gh ghops.GH, hostname string, activeLogin string, err error) error {
	if activeLogin == "" {
		return err
	}
	restoreErr := gh.SwitchUser(hostname, activeLogin)
	if err == nil {
		if restoreErr != nil {
			return fmt.Errorf("could not restore gh account %s: %w", activeLogin, restoreErr)
		}
		return nil
	}
	return ghops.RestoreResult(err, restoreErr, activeLogin)
}

func hasFlagKey(flags map[string]string, key string) bool {
	_, ok := flags[key]
	return ok
}
