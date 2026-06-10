# ghs

`ghs` is a small Go CLI for switching GitHub work/personal context without using root.

It manages separate concerns that `gh auth switch` does not change by itself:

- active GitHub CLI account: `gh auth switch`
- Git commit identity: `git config user.name/user.email`
- SSH key and host alias: `~/.ssh/config`
- repository remote URL: `git remote set-url origin ...`

## Install

**Requires Go 1.22+ and the [GitHub CLI](https://cli.github.com) (`gh`).**

```bash
go install github.com/izzamoe/ghs/cmd/ghs@latest
```

Verify:

```bash
ghs --help
```

> Make sure `$(go env GOPATH)/bin` is in your `PATH`. Add this to your shell profile if needed:
> ```bash
> export PATH="$PATH:$(go env GOPATH)/bin"
> ```

## Quick start

Import all your authenticated GitHub accounts as profiles:

```bash
ghs import-all
ghs list
```

Switch to a profile in the current repo:

```bash
ghs use me
```

## Add a profile

From the active GitHub CLI account:

```bash
ghs add-from-gh work --git-email work@example.com
```

`ghs add-from-gh` reads `gh api user`. GitHub may return an empty email when your profile email is private.
It also tries `gh api user/emails` and uses the primary verified email when your token has access. If that endpoint is blocked, refresh the scope:

```bash
gh auth refresh --scopes user:email
```

If an imported account still has no email, `ghs` falls back to the GitHub noreply address (`{id}+{login}@users.noreply.github.com`). To override it later:

```bash
ghs set-email work work@example.com
```

Until the email is filled, `ghs use work` only switches the active GitHub CLI account and does not change Git commit identity.

Manual profile creation is also supported:

```bash
ghs add-profile work \
  --gh-user zamyb \
  --git-name "IZZAMUDDIN ROYHUL FIRDAUS" \
  --git-email work@example.com \
  --ssh-alias github-work \
  --ssh-key ~/.ssh/id_ed25519_work \
  --workspace ~/Documents/work
```

Config is stored at `~/.config/ghs/config.conf`.

## Commands

```
ghs import-all                        # import every healthy GitHub CLI account on github.com
ghs import-all --no-overwrite         # import only new accounts, skip existing profiles
ghs list                              # list all profiles
ghs set-email <profile> <email>       # update git email for a profile
ghs use <profile>                     # switch gh account + local git identity
ghs use <profile> --global            # switch gh account + global git identity
ghs clone <profile> <owner/repo>      # clone via SSH alias and set local git identity
ghs clone <profile> <owner/repo> --upload-key  # also upload SSH public key to GitHub
ghs init-ssh <profile>                # generate SSH key if needed, write ~/.ssh/config block
ghs init-ssh <profile> --upload       # also upload SSH public key to GitHub
ghs fix-remote <profile>              # rewrite origin to git@<ssh-alias>:owner/repo.git
ghs status                            # show active gh account, git identity, and origin
```

### Notes

`ghs import-all` reads `gh auth status --hostname github.com --json hosts`, temporarily switches through each healthy account to read user details, saves one profile per login, then restores the previously active account.

`ghs clone <profile> <owner/repo>` switches `gh` to the profile account, ensures local SSH key/config exists, clones through the profile SSH alias, and sets git identity locally inside the cloned repo when the profile has an email.

Do not run `ghs` with `sudo` — root would write to `/root/.ssh` and `/root/.gitconfig`.
