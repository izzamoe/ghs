# ghs

`ghs` is a small Go CLI for switching GitHub work/personal context without using root.

It manages separate concerns that `gh auth switch` does not change by itself:

- active GitHub CLI account: `gh auth switch`
- Git commit identity: `git config user.name/user.email`, per repository, globally, or per directory (workspace)
- SSH key and host alias: `~/.ssh/config`
- repository remote URL: `git remote set-url origin ...`

`ghs` checks everything it can before it changes anything, rolls back what it
already changed when a later step fails, and names every file, account, and
remote it touched.

**Only github.com is supported.** GitHub Enterprise Server and other hosts are
rejected rather than half supported.

## Install

**Requires Go 1.26.3 or newer and the [GitHub CLI](https://cli.github.com) (`gh`).**
Git 2.30 or newer and an OpenSSH client are used for Git identity, workspaces,
and SSH checks.

```bash
go install github.com/izzamoe/ghs/cmd/ghs@latest
```

Verify:

```bash
ghs version   # prints the release tag, e.g. "ghs v0.5.0" (a local build prints "ghs devel")
ghs --help
```

Prebuilt binaries for Linux, macOS, and Windows (amd64 and arm64) are attached
to every [GitHub Release](https://github.com/izzamoe/ghs/releases).

> Make sure `$(go env GOPATH)/bin` is in your `PATH`. Add this to your shell profile if needed:
> ```bash
> export PATH="$PATH:$(go env GOPATH)/bin"
> ```

Do not run `ghs` with `sudo`: root would write to `/root/.ssh`, `/root/.gitconfig`,
and root's GitHub CLI state instead of yours. `ghs` never needs root.

## Quick start

Import all your authenticated GitHub accounts as profiles:

```bash
ghs import-all
ghs list
```

Switch to a profile in the current repo, and let `ghs` fix `origin` so pushes
use that profile's SSH key:

```bash
ghs use me --fix-remote
ghs doctor
```

## Commands

```
ghs add-profile <name> --gh-user <login> --git-name <name> --git-email <email> --ssh-alias <alias> --ssh-key <path> [--workspace <path>]
ghs add-from-gh <name> [--git-name <name>] [--git-email <email>] [--ssh-alias <alias>] [--ssh-key <path>] [--workspace <path>] [--require-email]
ghs import-all [--hostname github.com] [--require-email] [--no-overwrite]
ghs list
ghs status
ghs doctor [<profile>] [--offline]
ghs set-email <profile> <email>
ghs workspace <profile> <path>
ghs workspace <profile> --unlink
ghs use <profile> [--global] [--fix-remote]
ghs clone <profile> <owner/repo|github-url> [directory] [--upload-key]
ghs init-ssh <profile> [--upload]
ghs fix-remote <profile>
ghs remove <profile>
ghs version
ghs update
```

`ghs <command> --help` prints the command's flags. Flags may appear anywhere
after the command name, and `--` ends flag parsing. Unknown flags, missing or
empty values, repeated flags, values on boolean flags (`--global yes`), and
surplus arguments are rejected before anything runs.

Config is stored at `$XDG_CONFIG_HOME/ghs/config.conf`, or
`~/.config/ghs/config.conf` when `XDG_CONFIG_HOME` is unset. It is written
atomically with owner-only permissions; unknown keys are preserved.

### Exit codes

Every command uses the same exit code contract:

| Exit code | Meaning |
|-----------|---------|
| `0` | success (including `list`/`status` when the GitHub CLI is unavailable, and `doctor` with only `ok`, `warn`, or `skip`) |
| `1` | any runtime failure, including a failed rollback, or `doctor` with at least one `fail` |
| `2` | usage error: unknown command or flag, missing or repeated flag, value on a boolean flag, wrong number of arguments, or an invalid value on the command line |

Errors are printed to stderr as `ghs: <message>`; warnings as `ghs: warning: <message>`.

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

Until the email is filled, `ghs use work` only switches the active GitHub CLI account and says that Git identity was left unchanged.

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

Profile names and SSH aliases use letters, digits, `.`, `_`, and `-` (not
starting with `-`), are unique ignoring letter case, and cannot be a command
name; an alias cannot be `github.com`. Emails, logins, names, and key paths are
validated before anything is written.

`ghs import-all` reads `gh auth status --hostname github.com --json hosts`,
temporarily switches through each healthy account to read user details, saves
one profile per login, and always restores the previously active account.

## Switching: `use`

```bash
ghs use work               # inside a repository: gh account + local git identity
ghs use work --global      # gh account + global git identity
ghs use work --fix-remote  # also rewrite origin to git@github-work:owner/repo.git first
```

`ghs use` checks first that the profile's account is logged in and that you
are inside a repository (unless `--global`), then switches the account and
writes the identity. If the identity write fails, the previous account is
restored and both results are reported. With `--fix-remote`, `origin` is
rewritten first and restored if a later step fails; a non-GitHub or unknown
`origin` stops the command before anything changes.

Without `--fix-remote`, `ghs use` warns (exit code stays `0`) when `origin`
still uses `github.com` or another profile's alias, because pushes would not
use the profile's SSH key.

## Workspaces

A workspace makes every repository under a directory commit with a profile's
identity, without running `ghs use` in each one:

```bash
ghs workspace work ~/Documents/work
```

This writes a ghs-owned identity file `gitconfig-work` next to the config file,
adds one global Git entry
`includeIf.gitdir/i:~/Documents/work/.path = <config dir>/gitconfig-work` with
`git config --global --add` (`ghs` never edits `~/.gitconfig` by hand), and
saves the `workspace` key. Running it again changes nothing; a partial link is
completed. Two profiles cannot use the same or nested directories.

```bash
ghs workspace work --unlink
```

removes exactly that entry (`git config --global --unset --fixed-value`),
deletes the identity file, and drops the `workspace` key. Other `includeIf`
entries are left alone. `set-email`, `add-from-gh`, and `import-all` keep a
profile's workspace and refresh its identity file.

## Seeing where you are: `list`, `status`, `doctor`

```
$ ghs list
   PROFILE  GH USER     GIT EMAIL         SSH ALIAS    WORKSPACE         GH AUTH
*  work     zamyb-work  work@example.com  github-work  ~/Documents/work  active
   me       zamyb       me@example.com    github-me    -                 yes
   old      olduser     (missing)         github-old   -                 no
active gh account: zamyb-work (profile "work")
```

`*` marks the profile of the active GitHub CLI account; `GH AUTH` shows whether
each profile's login is `active`, logged in (`yes`), or not (`no`). Without the
GitHub CLI the column shows `?` and the command still succeeds.

```
$ ghs status
ghs status
----------
gh account:    zamyb-work (profile "work")
git identity:  Izzam <work@example.com> (profile "work", local)
origin:        git@github-me:acme/app.git (profile "me")
mismatch: origin resolves to profile "me" but gh account resolves to profile "work"
mismatch: origin resolves to profile "me" but git identity resolves to profile "work"
```

`status` resolves the active account, the Git identity (with its scope, and
`workspace` when it comes from a workspace identity file), and `origin` to
profiles, and prints a `mismatch:` line for every disagreement.

```
$ ghs doctor
ghs doctor: profile "work"
ok    gh: active account zamyb-work (profile "work")
ok    git identity: Izzam <work@example.com> (local, /repo/.git/config)
warn  origin: git@github.com:acme/app.git uses github.com directly; run: ghs fix-remote work
ok    ssh config: Host github-work -> github.com, IdentityFile /home/u/.ssh/id_ed25519_work
ok    ssh auth: github authenticated github-work as zamyb-work
skip  workspace: profile has no workspace
summary: 4 ok, 1 warn, 0 fail, 1 skip
```

`doctor` checks the profile of the active account (or the one you name) and
tells you the command that fixes each finding. It is read-only: its only
network access is `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>`,
which cannot prompt or write `known_hosts`; `--offline` skips it. It exits `1`
when any check fails.

## Remotes and cloning

`ghs fix-remote <profile>` rewrites `origin` to `git@<alias>:owner/repo.git`
only when it points at `github.com` (scp, `ssh://`, `https://`, or `http://`
form) or at another profile's alias. GitLab, GitHub Enterprise, and unknown SSH
hosts are refused; an `origin` already on the alias is left alone. It warns
when the alias has no `~/.ssh/config` block yet.

`ghs clone <profile> <owner/repo>` accepts only `owner/repo` or a `github.com`
URL, switches `gh` to the profile account (and stays switched), ensures the
SSH key and config block exist, clones through the profile SSH alias, and sets
the clone's local Git identity when the profile has an email.

`ghs init-ssh <profile>` generates the key if it does not exist and appends a
`Host` block to `~/.ssh/config` (quoting paths that contain spaces) unless one
already names the alias. With `--upload` it uploads the public key while the
profile's account is active and then switches back to the account you had.

## Removing a profile

```bash
ghs remove old
```

deletes the profile from the config (unlinking its workspace first). It never
deletes SSH keys, edits `~/.ssh/config`, logs accounts out, or changes Git
identities; instead it prints what was kept and the command to clean each up
by hand.

## Updating

`ghs update` runs `go install github.com/izzamoe/ghs/cmd/ghs@latest` and reports the version change. It requires `ghs` to have been installed via `go install` — local builds (`ghs version` prints `devel`) are not supported.

## Contributing

```bash
make check   # gofmt -l (must print nothing), go vet ./..., go test -race ./...
make e2e     # only the end-to-end suite
```

Without `make`: `gofmt -l . && go vet ./... && go test -race ./...`.

The test suite is hermetic: end-to-end tests in `internal/e2e` run the real
`ghs` binary against fake `gh`, `git`, `ssh`, and `ssh-keygen` executables in a
throwaway home directory with a minimal environment. They need no network and
never read or write your real `~/.ssh`, `~/.gitconfig`, or GitHub CLI state.
CI runs the same checks on Linux, macOS, and Windows with the Go version from
`go.mod`; a test fails when this README's Go version or command list drifts
from `go.mod` or `ghs --help`.

Releases: update `CHANGELOG.md`, then push a `vX.Y.Z` tag on `main`; the
release workflow publishes binaries and notes taken from the matching
changelog section.

## License

MIT; see [LICENSE](LICENSE).
