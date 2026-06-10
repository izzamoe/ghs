package cli

import (
	"cmp"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/ghops"
	"github.com/izzamoe/ghs/internal/runner"
)

var unsafeProfilePathChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func (a App) addProfile(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ghs add-profile <name> --gh-user <user> --git-name <name> --git-email <email> --ssh-alias <alias> --ssh-key <path>")
	}
	profile := config.Profile{Name: args[0]}
	flags := parseFlags(args[1:])
	profile.GitHubUser = flags["gh-user"]
	profile.GitName = flags["git-name"]
	profile.GitEmail = flags["git-email"]
	profile.SSHHostAlias = flags["ssh-alias"]
	profile.SSHKey = flags["ssh-key"]
	profile.Workspace = flags["workspace"]
	if err := validateCompleteProfile(profile); err != nil {
		return err
	}
	return a.saveProfile(profile)
}

func (a App) addFromGH(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: ghs add-from-gh <name> [--git-name <name>] [--git-email <email>] [--ssh-alias <alias>] [--ssh-key <path>]")
	}

	user, err := ghops.New(runner.New()).ActiveUser()
	if err != nil {
		return err
	}
	profile, err := profileFromGH(args[0], user, parseFlags(args[1:]))
	if err != nil {
		return err
	}
	if err := validateImportedProfile(profile); err != nil {
		return err
	}
	return a.saveProfile(profile)
}

func (a App) importAll(args []string) error {
	flags := parseFlags(args)
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
	if err := a.saveProfiles(profiles, hasFlagKey(flags, "no-overwrite")); err != nil {
		return restoreActive(gh, hostname, activeLogin, err)
	}

	restoreErr := restoreActive(gh, hostname, activeLogin, nil)
	if restoreErr != nil {
		return restoreErr
	}
	_, err = fmt.Fprintf(a.out, "imported %d profiles from gh host %q\n", len(profiles), hostname)

	return err
}

func (a App) saveProfile(profile config.Profile) error {
	if err := a.saveProfiles([]config.Profile{profile}, false); err != nil {
		return err
	}
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.out, "saved profile %q to %s\n", profile.Name, path)

	return err
}

func (a App) saveProfiles(profiles []config.Profile, noOverwrite bool) error {
	path, err := config.DefaultPath()
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		cfg = config.Config{}
	}
	for _, profile := range profiles {
		if idx := slices.IndexFunc(cfg.Profiles, func(p config.Profile) bool { return p.Name == profile.Name }); idx >= 0 {
			if !noOverwrite {
				cfg.Profiles[idx] = profile
			}
		} else {
			cfg.Profiles = append(cfg.Profiles, profile)
		}
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}

	return nil
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

func parseFlags(args []string) map[string]string {
	flags := make(map[string]string)
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") || len(arg) == 2 {
			continue
		}
		key := arg[2:]
		if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
			flags[key] = ""
			continue
		}
		flags[key] = args[i+1]
		i++
	}
	return flags
}
