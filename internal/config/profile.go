package config

import "strings"

type Config struct {
	Profiles []Profile
}

type Profile struct {
	Name         string
	GitHubUser   string
	GitName      string
	GitEmail     string
	SSHHostAlias string
	SSHKey       string
	Workspace    string
	// Extra holds keys this version does not know, in file order, so that
	// files written by newer versions survive a load and save unchanged.
	Extra []KeyValue
}

// KeyValue is one unknown config key and its value.
type KeyValue struct {
	Key   string
	Value string
}

// FindProfile looks a profile up by its exact name.
func (c Config) FindProfile(name string) (Profile, bool) {
	if i := c.Index(name); i >= 0 {
		return c.Profiles[i], true
	}
	return Profile{}, false
}

// Index returns the position of the profile with exactly this name, or -1.
func (c Config) Index(name string) int {
	for i, profile := range c.Profiles {
		if profile.Name == name {
			return i
		}
	}
	return -1
}

// FindProfileFold looks a profile up ignoring letter case; used to suggest
// the intended name after an exact lookup fails.
func (c Config) FindProfileFold(name string) (Profile, bool) {
	return c.find(func(p Profile) bool { return strings.EqualFold(p.Name, name) })
}

// ByLogin finds the profile whose gh_user is login (GitHub logins are
// case-insensitive).
func (c Config) ByLogin(login string) (Profile, bool) {
	if login == "" {
		return Profile{}, false
	}
	return c.find(func(p Profile) bool { return strings.EqualFold(p.GitHubUser, login) })
}

// ByEmail finds the profile whose git_email is email, ignoring case.
func (c Config) ByEmail(email string) (Profile, bool) {
	if email == "" {
		return Profile{}, false
	}
	return c.find(func(p Profile) bool { return strings.EqualFold(p.GitEmail, email) })
}

// ByAlias finds the profile whose SSH host alias is alias, ignoring case.
func (c Config) ByAlias(alias string) (Profile, bool) {
	if alias == "" {
		return Profile{}, false
	}
	return c.find(func(p Profile) bool { return strings.EqualFold(p.SSHHostAlias, alias) })
}

// ByWorkspace finds the profile linked to exactly this workspace directory.
func (c Config) ByWorkspace(path string) (Profile, bool) {
	want := NormalizeWorkspace(path)
	if want == "" {
		return Profile{}, false
	}
	return c.find(func(p Profile) bool {
		return p.Workspace != "" && strings.EqualFold(NormalizeWorkspace(p.Workspace), want)
	})
}

// Aliases lists every profile's SSH host alias.
func (c Config) Aliases() []string {
	out := make([]string, 0, len(c.Profiles))
	for _, p := range c.Profiles {
		if p.SSHHostAlias != "" {
			out = append(out, p.SSHHostAlias)
		}
	}
	return out
}

func (c Config) find(match func(Profile) bool) (Profile, bool) {
	for _, profile := range c.Profiles {
		if match(profile) {
			return profile, true
		}
	}
	return Profile{}, false
}
