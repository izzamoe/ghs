package config

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
}

func (c Config) FindProfile(name string) (Profile, bool) {
	for _, profile := range c.Profiles {
		if profile.Name == name {
			return profile, true
		}
	}
	return Profile{}, false
}
