package e2e

// Shared scenario fixtures. Profiles mirror the examples in
// contracts/cli.md: "work" (zamyb-work), "me" (zamyb), "old" (olduser, no
// email).

const cfgWork = `[work]
gh_user = "zamyb-work"
git_name = "Izzam"
git_email = "work@example.com"
ssh_host_alias = "github-work"
ssh_key = "~/.ssh/id_ed25519_work"
`

const cfgMe = `[me]
gh_user = "zamyb"
git_name = "Izzam"
git_email = "me@example.com"
ssh_host_alias = "github-me"
ssh_key = "~/.ssh/id_ed25519_me"
`

const cfgOld = `[old]
gh_user = "olduser"
git_name = "Old"
ssh_host_alias = "github-old"
ssh_key = "~/.ssh/id_ed25519_old"
`

// cfgAll is the three standard profiles in order.
const cfgAll = cfgWork + "\n" + cfgMe + "\n" + cfgOld

// seedAccounts makes every login a healthy github.com account; active (if
// non-empty) is the active one.
func (sb *sandbox) seedAccounts(active string, logins ...string) {
	sb.t.Helper()
	sb.setState(func(s *fakeState) {
		s.GH.Accounts = nil
		for _, l := range logins {
			s.GH.Accounts = append(s.GH.Accounts, ghAccount{Login: l, Active: l == active, State: "success"})
		}
	})
}

// seedStandard writes the three standard profiles and logs in zamyb-work
// (active) and zamyb.
func (sb *sandbox) seedStandard() {
	sb.t.Helper()
	sb.writeConfig(cfgAll)
	sb.seedAccounts("zamyb-work", "zamyb-work", "zamyb")
}

func addWorkArgs(extra ...string) []string {
	args := []string{"add-profile", "work", "--gh-user", "zamyb-work", "--git-name", "Izzam",
		"--git-email", "work@example.com", "--ssh-alias", "github-work", "--ssh-key", "~/.ssh/id_ed25519_work"}
	return append(args, extra...)
}

// sshBlock is the block ghs writes for an alias and an IdentityFile value.
func sshBlock(alias, identityFile string) string {
	return "Host " + alias + "\n  HostName github.com\n  User git\n  IdentityFile " + identityFile + "\n  IdentitiesOnly yes\n"
}
