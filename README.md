# ghs

[![ci](https://github.com/izzamoe/ghs/actions/workflows/ci.yml/badge.svg)](https://github.com/izzamoe/ghs/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/izzamoe/ghs.svg)](https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs)

Switch GitHub work/personal context safely: GitHub CLI account, Git identity,
SSH host alias, and `origin` remote in one command, without root.

## Contents

- [What ghs does](#what-ghs-does)
- [Install](#install)
- [First run](#first-run)
- [Commands](#commands)
- [Command reference](#command-reference)
- [Profiles](#profiles)
- [Switching: `use`](#switching-use)
- [Workspaces](#workspaces)
- [Seeing where you are: `list`, `status`, `doctor`](#seeing-where-you-are-list-status-doctor)
- [Remotes and cloning](#remotes-and-cloning)
- [Removing a profile](#removing-a-profile)
- [Troubleshooting](#troubleshooting)
- [Config and data](#config-and-data)
- [Cross-platform notes](#cross-platform-notes)
- [Contributing](#contributing)
- [Security](#security)
- [Support](#support)
- [License](#license)

## What ghs does

`ghs` is a small Go CLI for people with more than one GitHub account, for
example a personal and a work account. It manages the separate things that
`gh auth switch` does not change by itself:

- the active GitHub CLI account: `gh auth switch`
- the Git commit identity: `git config user.name/user.email`, per repository,
  globally, or per directory (workspace)
- the SSH key and host alias: a `Host` block in `~/.ssh/config`
- the repository remote: `git remote set-url origin ...`

`ghs` checks everything it can before it changes anything, rolls back what it
already changed when a later step fails, and names every file, account, and
remote it touched. It never reads private keys and stores no tokens.

**Only github.com is supported.** GitHub Enterprise Server and other hosts are
rejected rather than half supported.

Full documentation: https://github.com/izzamoe/ghs#readme (this file; release
archives include a copy).

## Install

### Requirements

| Tool | Needed for | Version |
|------|------------|---------|
| [GitHub CLI](https://cli.github.com) (`gh`) | account switching, importing profiles, uploading keys | a release whose `gh auth status` supports `--json` |
| Git | identities, workspaces, remotes, `clone` | 2.30 or newer (`ghs` uses `git config --fixed-value`) |
| OpenSSH client (`ssh`, `ssh-keygen`) | generating keys (`init-ssh`, `clone`), the `doctor` SSH check, and your pushes | any current release |
| Go | only for Option A and for `ghs update` | see Option A |

`ghs` itself is a single binary and needs nothing beyond these tools.

### Option A: Go toolchain

Requires Go 1.26.3 or newer. With Go 1.21 or newer and the default
`GOTOOLCHAIN=auto`, an older local Go downloads the required toolchain by
itself.

```bash
go install github.com/izzamoe/ghs/cmd/ghs@latest
```

`go install` puts the binary in `$(go env GOPATH)/bin` (or `$GOBIN` when set).
Make sure that directory is on your `PATH`; for example, add this to your
shell profile:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

On Windows the default directory is `%USERPROFILE%\go\bin`; add it to your
user `PATH` if `ghs` is not found.

### Option B: Release archive

Every [GitHub release](https://github.com/izzamoe/ghs/releases/latest) has
prebuilt binaries and a `checksums.txt` file. Archives are named
`ghs_<version>_<os>_<arch>.tar.gz`, or `ghs_<version>_windows_<arch>.zip` on
Windows, where `<version>` is the release without its leading `v` (for
example `0.5.0`):

| OS | CPU | Archive |
|----|-----|---------|
| Linux | x86-64 | `ghs_<version>_linux_amd64.tar.gz` |
| Linux | ARM64 | `ghs_<version>_linux_arm64.tar.gz` |
| macOS | Intel | `ghs_<version>_darwin_amd64.tar.gz` |
| macOS | Apple Silicon | `ghs_<version>_darwin_arm64.tar.gz` |
| Windows | x86-64 | `ghs_<version>_windows_amd64.zip` |
| Windows | ARM64 | `ghs_<version>_windows_arm64.zip` |

Each archive contains the `ghs` binary (`ghs.exe` on Windows), `README.md`,
`LICENSE`, and `CHANGELOG.md`.

1. Download your archive and `checksums.txt` from
   https://github.com/izzamoe/ghs/releases/latest, in a browser or with the
   GitHub CLI you need anyway:

   ```bash
   gh release download --repo izzamoe/ghs --pattern 'ghs_*_linux_amd64.tar.gz' --pattern checksums.txt
   ```

2. Verify the checksum.

   Linux (must print `OK` for your archive):

   ```bash
   sha256sum --check --ignore-missing checksums.txt
   ```

   macOS (replace the archive name with yours; the two hashes must be equal):

   ```bash
   shasum -a 256 ghs_0.5.0_darwin_arm64.tar.gz
   grep ghs_0.5.0_darwin_arm64.tar.gz checksums.txt
   ```

   Windows PowerShell (the two values must be equal):

   ```powershell
   (Get-FileHash .\ghs_0.5.0_windows_amd64.zip -Algorithm SHA256).Hash.ToLower()
   Select-String ghs_0.5.0_windows_amd64.zip .\checksums.txt
   ```

3. Extract the binary and put it in a directory on your `PATH`:

   ```bash
   tar -xzf ghs_0.5.0_linux_amd64.tar.gz ghs
   mkdir -p ~/.local/bin && mv ghs ~/.local/bin/    # or any directory on PATH
   ```

   ```powershell
   Expand-Archive .\ghs_0.5.0_windows_amd64.zip -DestinationPath "$env:LOCALAPPDATA\Programs\ghs"
   # then add $env:LOCALAPPDATA\Programs\ghs to your user PATH
   ```

4. macOS: the binaries are not signed or notarized. A file downloaded with a
   browser may be blocked by Gatekeeper; removing the quarantine attribute
   may be needed: `xattr -d com.apple.quarantine ghs`.

ghs is not distributed through Homebrew, Scoop, winget, apt, or the AUR; the two options above are the only supported install paths.

### Verify the install

```bash
ghs version   # prints the release tag, e.g. "ghs v0.5.0" (a local build prints "ghs devel")
ghs --help
```

### Do not use sudo

Do not run `ghs` with `sudo`: root would write to `/root/.ssh`, `/root/.gitconfig`,
and root's GitHub CLI state instead of yours. `ghs` never needs root. If it
already happened, see [Troubleshooting](#troubleshooting).

### Updating

- Option A: `ghs update` runs `go install github.com/izzamoe/ghs/cmd/ghs@latest`
  and reports the version change (`updated ghs v0.5.0 → v0.5.1`, or
  `ghs v0.5.0 is already up to date`). It needs `go` on `PATH`. It refuses
  local builds (`ghs version` prints `devel`); reinstall those with the
  `go install` command.
- Option B: download, verify, and extract the new archive the same way,
  replacing the old binary. Do not use `ghs update` for an archive install:
  it would install a second copy into `$(go env GOPATH)/bin` (if Go is
  installed) and leave the binary you run unchanged.

Run `ghs version` before and after to confirm.

### Uninstall

1. Delete the binary: `$(go env GOPATH)/bin/ghs` for Option A, or wherever you
   put it for Option B.
2. Delete the config directory, `$XDG_CONFIG_HOME/ghs` or `~/.config/ghs`
   (`%USERPROFILE%\.config\ghs` on Windows). This also removes the
   `gitconfig-<profile>` identity files used by workspaces; unlink workspaces
   first (`ghs workspace <profile> --unlink`) or remove their `includeIf`
   entries by hand, see below.

`ghs` never deletes these for you; they stay until you remove them:

- SSH keys it generated (`~/.ssh/id_ed25519_<name>` and `.pub`), and public
  keys it uploaded to your GitHub accounts;
- the `Host <alias>` blocks it appended to `~/.ssh/config`;
- global `includeIf` entries for workspaces: list them with
  `git config --global --get-regexp includeIf`, remove one with
  `git config --global --unset --fixed-value 'includeIf.gitdir/i:<dir>/.path' <file>`;
- local and global `user.name` / `user.email` values it set;
- `origin` URLs it rewrote to an SSH alias;
- your GitHub CLI logins.

## First run

This walk-through uses two accounts, `alice` (personal) and `alice-work`.
Output is shortened to the lines `ghs` prints; paths depend on your system.

1. Log in to every account with the GitHub CLI, one after the other, and
   check that each one is listed:

   ```bash
   gh auth login --hostname github.com    # once per account
   gh auth status --hostname github.com
   ```

2. Create one profile per logged-in account. Profiles are named after the
   GitHub login; the SSH alias becomes `github-<login>` and the key
   `~/.ssh/id_ed25519_<login>`. The previously active account is restored
   afterwards.

   ```
   $ ghs import-all
   imported 2 profiles from gh host "github.com"
   ```

   If an account's email is private, `ghs` falls back to its noreply address;
   see [Profiles](#profiles) to set a different one.

3. Check the profiles:

   ```
   $ ghs list
      PROFILE     GH USER     GIT EMAIL               SSH ALIAS          WORKSPACE  GH AUTH
   *  alice       alice       alice@example.com       github-alice       -          active
      alice-work  alice-work  alice@work.example.com  github-alice-work  -          yes
   active gh account: alice (profile "alice")
   ```

4. Create each profile's SSH key and alias, and register the public key on
   the right account (`ghs` switches to that account for the upload and back
   afterwards). Existing keys and `Host` blocks are reused, never replaced.

   ```
   $ ghs init-ssh alice-work --upload
   generated ssh key /home/alice/.ssh/id_ed25519_alice-work
   appended Host github-alice-work to /home/alice/.ssh/config
   ssh is ready for profile "alice-work" via host "github-alice-work"
   uploaded public key /home/alice/.ssh/id_ed25519_alice-work.pub to account alice-work
   ```

   Repeat for `alice`. Running it again prints
   `public key ... already registered on account alice-work`.

5. Inside a repository, switch to a profile and route `origin` through its
   alias so pushes use its key:

   ```
   $ cd ~/src/app
   $ ghs use alice-work --fix-remote
   origin updated: git@github.com:acme/app.git -> git@github-alice-work:acme/app.git
   switched gh account: alice -> alice-work
   git identity set: Alice Example <alice@work.example.com> (local: /home/alice/src/app)
   ```

6. Confirm everything agrees. `doctor` is read-only; `--offline` skips its
   only network check.

   ```
   $ ghs doctor
   ghs doctor: profile "alice-work"
   ok    gh: active account alice-work (profile "alice-work")
   ok    git identity: Alice Example <alice@work.example.com> (local, /home/alice/src/app/.git/config)
   ok    origin: git@github-alice-work:acme/app.git uses alias github-alice-work
   ok    ssh config: Host github-alice-work -> github.com, IdentityFile /home/alice/.ssh/id_ed25519_alice-work
   ok    ssh auth: github authenticated github-alice-work as alice-work
   skip  workspace: profile has no workspace
   summary: 5 ok, 0 warn, 0 fail, 1 skip
   ```

   A `fail` line names the command that fixes it. `ssh auth` fails with
   `Host key verification failed` the first time; see
   [Troubleshooting](#troubleshooting).

7. Optional: make every repository under a directory commit as a profile,
   without running `ghs use` in each one:

   ```
   $ ghs workspace alice-work ~/work
   wrote identity file /home/alice/.config/ghs/gitconfig-alice-work
   added git include: includeIf.gitdir/i:~/work/.path = /home/alice/.config/ghs/gitconfig-alice-work
   saved workspace ~/work for profile "alice-work"
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

`ghs <command> --help` and `ghs help <command>` print a command's usage and
flags; `ghs --help` prints this list. Every command is documented in full
under [Command reference](#command-reference).

Config is stored at `$XDG_CONFIG_HOME/ghs/config.conf`, or
`~/.config/ghs/config.conf` when `XDG_CONFIG_HOME` is unset; see
[Config and data](#config-and-data).

### Exit codes

Every command uses the same exit code contract:

| Exit code | Meaning |
|-----------|---------|
| `0` | success (including `list`/`status` when the GitHub CLI is unavailable, and `doctor` with only `ok`, `warn`, or `skip`) |
| `1` | any runtime failure, including a failed rollback, or `doctor` with at least one `fail` |
| `2` | usage error: unknown command or flag, missing or repeated flag, value on a boolean flag, wrong number of arguments, or an invalid value on the command line |

Success output goes to stdout. Errors go to stderr as `ghs: <message>`, and
warnings as `ghs: warning: <message>`; a warning never changes the exit code.

### Flag rules

Flags may appear anywhere after the command name, and `--` ends flag parsing
(`ghs set-email work -- new@example.com`). A flag that takes a value needs a non-empty value
(`--git-email a@example.com` or `--git-email=a@example.com`). Unknown flags,
missing or empty values, repeated flags, values on boolean flags
(`--global yes`), and surplus arguments are rejected with exit code `2`
before anything runs; nothing is silently ignored.

## Command reference

Each block below is exactly what `ghs <command> --help` (or
`ghs help <command>`) prints.

### `ghs add-profile`

```
ghs add-profile <name> --gh-user <login> --git-name <name> --git-email <email> --ssh-alias <alias> --ssh-key <path> [--workspace <path>]
  --gh-user <login>      GitHub login of the account (required)
  --git-name <name>      Git author name (required)
  --git-email <email>    Git author email (required)
  --ssh-alias <alias>    SSH host alias, e.g. github-work (required)
  --ssh-key <path>       SSH private key path, absolute or ~/ (required)
  --workspace <path>     link this directory to the profile's Git identity
```

### `ghs add-from-gh`

```
ghs add-from-gh <name> [--git-name <name>] [--git-email <email>] [--ssh-alias <alias>] [--ssh-key <path>] [--workspace <path>] [--require-email]
  --git-name <name>      Git author name (default: the GitHub name)
  --git-email <email>    Git author email (default: the GitHub email or noreply address)
  --ssh-alias <alias>    SSH host alias (default: github-<name>)
  --ssh-key <path>       SSH private key path (default: ~/.ssh/id_ed25519_<name>)
  --workspace <path>     link this directory to the profile's Git identity
  --require-email        fail instead of saving a profile without an email
```

### `ghs import-all`

```
ghs import-all [--hostname github.com] [--require-email] [--no-overwrite]
  --hostname github.com  GitHub CLI host; only github.com is supported
  --require-email        fail when an account has no email
  --no-overwrite         keep existing profiles with the same name
```

### `ghs list`

```
ghs list
  (ghs list takes no flags)
```

### `ghs status`

```
ghs status
  (ghs status takes no flags)
```

### `ghs doctor`

```
ghs doctor [<profile>] [--offline]
  --offline              skip the ssh -T authentication check (no network)
```

### `ghs set-email`

```
ghs set-email <profile> <email>
  (ghs set-email takes no flags)
```

### `ghs workspace`

```
ghs workspace <profile> <path>
ghs workspace <profile> --unlink
  --unlink               remove the profile's workspace link
```

### `ghs use`

```
ghs use <profile> [--global] [--fix-remote]
  --global               set the global Git identity instead of the repository's
  --fix-remote           first rewrite origin to go through the profile's SSH alias
```

### `ghs clone`

```
ghs clone <profile> <owner/repo|github-url> [directory] [--upload-key]
  --upload-key           upload the SSH public key to the profile's account
```

### `ghs init-ssh`

```
ghs init-ssh <profile> [--upload]
  --upload               upload the SSH public key to the profile's account
```

### `ghs fix-remote`

```
ghs fix-remote <profile>
  (ghs fix-remote takes no flags)
```

### `ghs remove`

```
ghs remove <profile>
  (ghs remove takes no flags)
```

### `ghs version`

```
ghs version
  (ghs version takes no flags)
```

### `ghs update`

```
ghs update
  (ghs update takes no flags)
```

## Profiles

A profile ties one GitHub account to a Git identity, an SSH alias, and a key.
The examples below use a profile `work` for the account `alice-work` and a
profile `me` for `alice`; `ghs import-all` (see [First run](#first-run))
names profiles after the login instead.

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
  --gh-user alice-work \
  --git-name "Alice Example" \
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
*  work     alice-work  work@example.com  github-work  ~/Documents/work  active
   me       alice       me@example.com    github-me    -                 yes
   old      olduser     (missing)         github-old   -                 no
active gh account: alice-work (profile "work")
```

`*` marks the profile of the active GitHub CLI account; `GH AUTH` shows whether
each profile's login is `active`, logged in (`yes`), or not (`no`). Without the
GitHub CLI the column shows `?` and the command still succeeds.

```
$ ghs status
ghs status
----------
gh account:    alice-work (profile "work")
git identity:  Alice Example <work@example.com> (profile "work", local)
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
ok    gh: active account alice-work (profile "work")
ok    git identity: Alice Example <work@example.com> (local, /repo/.git/config)
warn  origin: git@github.com:acme/app.git uses github.com directly; run: ghs fix-remote work
ok    ssh config: Host github-work -> github.com, IdentityFile /home/alice/.ssh/id_ed25519_work
ok    ssh auth: github authenticated github-work as alice-work
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

## Troubleshooting

Start with `ghs doctor` (or `ghs doctor <profile>`), then `ghs status`.
`doctor` runs six read-only checks in this order and prints one line each:

- `gh`: the GitHub CLI is installed and the active account is the profile's;
- `git identity`: `user.email` (and `user.name`) in effect here match the
  profile, with the scope and file they come from;
- `origin`: the remote uses the profile's alias rather than `github.com` or
  another profile's alias;
- `ssh config`: `~/.ssh/config` has a complete `Host <alias>` block whose key
  pair exists;
- `ssh auth`: GitHub greets the alias as the profile's login (skipped with
  `--offline`);
- `workspace`: a configured workspace is linked and its identity file is
  current.

Each line starts with an outcome: `ok` (as expected), `warn` (works, but
probably not what you want), `fail` (broken; `doctor` exits `1`), or `skip`
(not applicable here, or `--offline`). Every `warn` and `fail` line ends with
the command that fixes it.

### Symptoms and fixes

Every fix has a manual equivalent that needs only `gh`, `git`, `ssh`, and a
text editor, so you can recover even when `ghs` cannot run.

| Symptom | Cause | Fix with ghs | Fix by hand |
|---------|-------|--------------|-------------|
| `git push` fails with `Permission denied (publickey)` | `origin` uses `github.com` or an alias whose key is not on the account | `ghs doctor <profile>`: a `fail` on `ssh config` or `ssh auth` means the key or block is missing, then `ghs init-ssh <profile> --upload` (adds only this profile's key and block; other keys are untouched); a `warn` on `origin` means `ghs fix-remote <profile>` | `ssh -T git@<alias>` to see which account the key belongs to; add `<key>.pub` at https://github.com/settings/keys; `git remote set-url origin git@<alias>:<owner>/<repo>.git` |
| `doctor` shows `ssh auth: Host key verification failed.` | `doctor` runs `ssh` in batch mode and never writes `known_hosts`, so GitHub's host key is not trusted yet | none, by design | run `ssh -T git@<alias>` once in a terminal, compare the fingerprint with [GitHub's SSH key fingerprints](https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/githubs-ssh-key-fingerprints), and accept it |
| `ghs init-ssh --upload` or `ghs clone --upload-key` fails with a `gh ssh-key add` error about a missing scope | the account's GitHub CLI token cannot manage SSH keys | switch to the account (`gh auth switch --user <login>`), run `gh auth refresh --hostname github.com --scopes admin:public_key`, then retry | add `<key>.pub` at https://github.com/settings/keys while signed in as that account |
| a push, pull request, or issue went to the wrong account | the active `gh` account, the Git identity, and `origin` resolve to different profiles | `ghs status` shows which one disagrees; then `ghs use <profile> --fix-remote` (or `ghs fix-remote <profile>`) | `gh auth switch --user <login>`; `git remote set-url origin git@<alias>:<owner>/<repo>.git` |
| the last commit has the wrong author email | the identity was set for another profile, or not at all | `ghs use <profile>` (or link a workspace with `ghs workspace`), then `git commit --amend --reset-author` | `git config user.email <email>` and `git config user.name "<name>"`, then `git commit --amend --reset-author` |
| `ghs use` failed and printed `restored origin to <url>` and/or `restored gh account <login>` | a later step failed; `ghs` already undid the earlier ones | read the first `ghs:` line for the cause, fix it, and rerun | nothing to undo |
| `ghs use` printed `could not restore origin to <url>` or `could not restore gh account <login>` | the rollback itself failed | fix the cause named on that line, then run the manual commands | `git remote set-url origin <url>` and `gh auth switch --user <login>`, using the URL and login from the `could not restore` lines |
| `ghs: invalid config <path>: line <n>: ...` or `ghs: cannot read config <path>: ...` | the config file cannot be read or parsed; `ghs` refuses to overwrite it and stops | none, by design | fix line `<n>` of the named file in an editor, or move the file aside and run `ghs import-all` again |
| `profile "Work" not found; did you mean "work"?` | profile names are exact; the lookup suggests a letter-case variant | use the suggested name | none |
| `git identity unchanged: profile "<name>" has no email` | the profile has no `git_email` (GitHub returned none, or the config was edited) | `ghs set-email <profile> <email>`, or `gh auth refresh --scopes user:email` and import again | set `git_email` for the profile in the config file |
| `gh account <login> is not logged in for github.com` | the GitHub CLI has no valid token for that login | `gh auth login --hostname github.com` | same |
| `origin ... is not a GitHub remote` or `... is neither github.com nor the alias of a ghs profile` | `ghs` rewrites only github.com URLs and known aliases | none: `ghs fix-remote` refuses by design | `git remote set-url origin <url>` yourself |
| `update is not available for local builds` or `update is only available when installed via go install` | the binary was built from source or came from a release archive | `go install github.com/izzamoe/ghs/cmd/ghs@latest`, or download the next archive | same |
| you ran `ghs` with `sudo` | files were created as root, under root's home (`/root` on Linux) or owned by root inside yours | none | remove what root created: `sudo rm -r /root/.config/ghs`, the `Host` blocks in `/root/.ssh/config`, root's global `includeIf` and `user.*` entries (`sudo git config --global --list` shows them); root's own GitHub CLI login with `sudo gh auth logout --hostname github.com` (this does not touch your logins); find files owned by root in your home with `find ~/.config/ghs ~/.ssh ~/.gitconfig -user root` and fix them with `sudo chown "$(id -un)" <file>`, including `.git/config` of a repository you used; then redo [First run](#first-run) as yourself |
| Windows: `doctor` shows `ssh auth: ssh is not installed`, or `init-ssh` cannot run `ssh-keygen` | the OpenSSH client is not installed or not on `PATH` | none | install "OpenSSH Client" (Settings → System → Optional features on Windows 11, Settings → Apps → Optional features on Windows 10) or use the OpenSSH that comes with Git for Windows, and make sure `ssh` and `ssh-keygen` are on `PATH` |

### What ghs changes and how to undo it

Every change is printed when it happens (for example
`origin updated: <old> -> <new>` or `switched gh account: <old> -> <new>`),
so the old value you need for an undo is in the output.

| What changes | Commands that change it | How it is written | Undo by hand |
|--------------|-------------------------|-------------------|--------------|
| `config.conf` | `add-profile`, `add-from-gh`, `import-all`, `set-email`, `workspace`, `remove` | atomically (temporary file, then rename), mode `0600` | edit or delete the file; `ghs` never overwrites a file it cannot parse |
| identity file `gitconfig-<profile>` next to the config | `workspace`, and refreshed by `set-email`, `add-from-gh`, `import-all` | atomically, mode `0600` | remove its `includeIf` entry (below), then delete the file |
| `~/.ssh/config` | `init-ssh`, `clone` | one `Host <alias>` block appended; the file is never rewritten | delete the block in an editor |
| SSH key pair `<key>` and `<key>.pub` | `init-ssh`, `clone` | `ssh-keygen -t ed25519`; an existing key is never overwritten and `ghs` never deletes one | nothing is required; the key is yours. If you no longer use it, remove it from the account first (row below), then delete both files |
| global `includeIf.gitdir/i:<dir>/.path` | `workspace`, and `add-profile`/`add-from-gh` with `--workspace` | `git config --global --add` | `ghs workspace <profile> --unlink`, or `git config --global --unset --fixed-value 'includeIf.gitdir/i:<dir>/.path' <file>` |
| repository or global `user.name` / `user.email` | `use` (`--global` for global), `clone` (in the new clone) | `git config [--global]` | `git config [--global] user.email <old>`, or `git config [--global] --unset user.email` (same for `user.name`) |
| `origin` URL | `use --fix-remote`, `fix-remote` | `git remote set-url origin` | `git remote set-url origin <old>` with the old URL from `origin updated: <old> -> <new>` |
| active GitHub CLI account | `use` and `clone` (stays switched); `import-all` and `init-ssh --upload` (switch back afterwards) | `gh auth switch` | `gh auth switch --user <login>` |
| public key on your GitHub account | `init-ssh --upload`, `clone --upload-key` | `gh ssh-key add <key>.pub --title ghs-<profile>` (only the `.pub` file) | `gh ssh-key list`, then `gh ssh-key delete <id>` with that account active, or delete it at https://github.com/settings/keys |
| a new directory | `clone` | `git clone` through the alias | delete the directory |

## Config and data

### Files ghs writes

| File | Mode | How it is written |
|------|------|-------------------|
| `$XDG_CONFIG_HOME/ghs/config.conf`, or `~/.config/ghs/config.conf` when `XDG_CONFIG_HOME` is unset | `0600` for a new file (an existing file keeps its mode); the directory is created `0700` | atomically: a temporary file in the same directory, flushed, then renamed over the old one; a failed write leaves the old file untouched |
| `<config dir>/gitconfig-<profile>` (workspace identity file with `[user] name` and `email`) | `0600` | atomically, as above |
| `~/.ssh/config` | `0600` when `ghs` creates it; `~/.ssh` is created `0700` | append-only: one `Host` block per alias, only when no `Host` line names the alias yet |
| `~/.ssh/<key>` and `<key>.pub` | as created by `ssh-keygen` | `ssh-keygen -t ed25519 -C <login> -f <key> -N ""`, only when the key does not exist |
| Git config (repository `.git/config`, global `~/.gitconfig`) | unchanged | only through `git config` and `git remote set-url`; `ghs` never edits these files itself |

The SSH block `ghs` appends looks like this (`IdentityFile` uses forward
slashes and is quoted when the path contains spaces):

```
Host github-work
  HostName github.com
  User git
  IdentityFile /home/alice/.ssh/id_ed25519_work
  IdentitiesOnly yes
```

### Files ghs reads

- the config file and the `gitconfig-<profile>` identity files;
- `~/.ssh/config`, to find existing `Host` blocks;
- `<key>.pub`, only to upload it; `ghs` checks that the private key file
  exists but never opens it;
- Git configuration, through `git config` (for example
  `git config --show-origin --show-scope` in `status` and `doctor`);
- the GitHub CLI's account list, through
  `gh auth status --hostname github.com --json hosts`.

### Network access

`ghs` opens no network connections itself. These subprocesses may contact
GitHub, and only when the listed command runs:

| Subprocess | Triggered by |
|------------|--------------|
| `gh auth status --hostname github.com --json hosts` (the GitHub CLI may validate its tokens online) | `use`, `clone`, `init-ssh --upload`, `import-all`, `list`, `status`, `doctor`, `remove` |
| `gh api user` and `gh api user/emails` | `add-from-gh`, `import-all` |
| `gh ssh-key add <key>.pub --title ghs-<profile>` | `init-ssh --upload`, `clone --upload-key` |
| `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` | `doctor` without `--offline` |
| `git clone git@<alias>:<owner>/<repo>.git` | `clone` |
| `go install github.com/izzamoe/ghs/cmd/ghs@latest` | `update` |

`gh auth switch --hostname github.com --user <login>` (run by `use`,
`clone`, `import-all`, and `init-ssh --upload`) only changes the GitHub CLI's
local state.

### What ghs never does

- It never reads, prints, or uploads private key material; only the `.pub`
  file is passed to `gh ssh-key add`.
- It stores no tokens or passwords; the GitHub CLI keeps its own credentials.
- It sends no telemetry and has no automatic update check.
- It never runs a shell: `gh`, `git`, `ssh`, `ssh-keygen`, and `go` are run
  with argument lists.
- It never deletes SSH keys, rewrites `~/.ssh/config`, edits `~/.gitconfig`
  by hand, or logs a GitHub CLI account out.

### Config file format

The config is an INI-like text file with one section per profile. This is a
complete profile as `ghs` writes it:

```ini
[work]
gh_user = "alice-work"
git_name = "Alice Example"
git_email = "alice@example.com"
ssh_host_alias = "github-work"
ssh_key = "~/.ssh/id_ed25519_work"
workspace = "~/Documents/work"
```

Keys `ghs` does not know are kept, in order, when it saves the file, so newer
config files do not break older binaries. Lines starting with `#` or `;` are
accepted when reading, but comments and blank lines are not kept when `ghs`
saves. The file is safe to edit by hand while `ghs` is not running; if an
edit breaks it, every command that reads the config stops with the file and
line number instead of overwriting it.

## Cross-platform notes

### Linux

The config lives in `$XDG_CONFIG_HOME/ghs/` when `XDG_CONFIG_HOME` is set,
otherwise in `~/.config/ghs/`. Nothing else is platform-specific. Install
`gh`, `git`, and `openssh` from your distribution or from their projects.

### macOS

Apple's bundled `git` (from the Command Line Tools) and OpenSSH work. `ghs`
does not add keys to the macOS keychain or to `ssh-agent`; the generated
`Host` block names the key with `IdentityFile`, which is all `ssh` needs.
Both Intel (`darwin_amd64`) and Apple Silicon (`darwin_arm64`) archives are
published; see the Gatekeeper note under
[Option B](#option-b-release-archive).

### Windows

- Paths: the config is `%USERPROFILE%\.config\ghs\config.conf` (or under
  `XDG_CONFIG_HOME` when set), the SSH config is `%USERPROFILE%\.ssh\config`,
  and keys go to `%USERPROFILE%\.ssh\`.
- The OpenSSH client must be installed: "OpenSSH Client" under Optional
  features in Settings, or the one that comes with Git for Windows. `ssh` and `ssh-keygen` must be on `PATH`.
- Key and workspace paths may be given with backslashes; they are
  normalized. `IdentityFile` is written with forward slashes, which OpenSSH on
  Windows accepts, and quoted when the path contains spaces.
- Files are written with LF line endings.
- Release archives are `.zip` files; verify them with `Get-FileHash` as shown
  under [Option B](#option-b-release-archive).
- `make` is not needed: every gate has a plain command in
  [CONTRIBUTING.md](CONTRIBUTING.md).
- `ghs update` needs `go` on `PATH`.

## Contributing

Gates: `make check` (`gofmt -l`, `go vet ./...`, `go test -race ./...`) and
`make e2e` for only the end-to-end suite; without `make`, run
`gofmt -l . && go vet ./... && go test -race ./...`. The test suite is
hermetic: the end-to-end tests in `internal/e2e` run the real `ghs` binary
against fake `gh`, `git`, `ssh`, and `ssh-keygen` in a throwaway home
directory, need no network, and never touch your real `~/.ssh`,
`~/.gitconfig`, or GitHub CLI state. Tests also fail when this README drifts
from `ghs --help`, `go.mod`, `.goreleaser.yaml`, or the `Makefile`.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the full guide and the maintainer
release runbook, and [CHANGELOG.md](CHANGELOG.md) for what changed in each
release.

## Security

Please report vulnerabilities privately as described in
[SECURITY.md](SECURITY.md); do not open a public issue with details.

## Support

See [SUPPORT.md](SUPPORT.md) for self-help steps and how to ask a question.
Support is best effort by one maintainer. Everyone taking part follows the
[Code of Conduct](CODE_OF_CONDUCT.md).

## License

MIT; see [LICENSE](LICENSE).
