# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).
While the major version is `0`, a minor release may contain breaking CLI
changes; they are marked **Breaking:** below.

## [Unreleased]

## [0.5.1] - 2026-09-28

### Fixed

- Publish the account-context safety release after the initial `v0.5.0` tag
  could not generate notes because its changelog section was absent.

## [0.5.0] - 2026-09-28

### Added

- `ghs remove <profile>` deletes one profile from the config without touching
  SSH keys, `~/.ssh/config`, GitHub CLI logins, or Git identities, and lists
  what was kept with the manual cleanup command for each.
- `ghs doctor [<profile>] [--offline]` runs six read-only checks (`gh`,
  `git identity`, `origin`, `ssh config`, `ssh auth`, `workspace`), prints
  `ok`/`warn`/`fail`/`skip` with the remedy for each finding, and exits `1`
  when any check fails. The SSH check runs
  `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` and never writes
  `known_hosts`; `--offline` skips it.
- `ghs workspace <profile> <path>` links a directory to a profile's Git
  identity through a ghs-owned identity file and a global
  `includeIf.gitdir/i:<path>/.path` entry added with `git config --global`;
  `ghs workspace <profile> --unlink` removes exactly that entry. The
  `--workspace` flag of `add-profile` and `add-from-gh` now performs the link.
- `ghs use <profile> --fix-remote` rewrites `origin` to the profile's SSH
  alias before switching, and rolls it back if a later step fails.
- `ghs use` prints a warning when `origin` still uses `github.com` or another
  profile's alias.
- `ghs list` marks the active profile with `*` and adds `WORKSPACE` and
  `GH AUTH` columns plus an `active gh account:` line.
- `ghs status` resolves the active account, the Git identity (with its scope
  and `workspace` origin), and `origin` to profiles and prints a `mismatch:`
  line for every disagreement.
- `ghs fix-remote` warns when the profile's alias has no `~/.ssh/config`
  block yet (the rewrite still happens).
- `ghs workspace` refuses to link a profile that is already linked to a
  different directory until it is unlinked, so no stale `includeIf` entry is
  left behind.
- `ghs <command> --help` prints that command's usage.
- MIT `LICENSE`, this changelog, CI on Linux, macOS, and Windows, and a
  tag-driven release that publishes binaries for six platform/architecture
  pairs with notes taken from this file.

### Changed

- **Breaking:** flags are parsed strictly. Unknown flags, missing or empty
  values, repeated flags, values on boolean flags (`--global yes`), and
  surplus or missing arguments are usage errors that exit with code `2`
  before any tool runs. Flags may appear anywhere after the command; `--`
  ends flag parsing.
- **Breaking:** exit codes are `0` success, `1` failure, `2` usage error.
- **Breaking:** `import-all --hostname` accepts only `github.com`; other hosts
  are rejected instead of producing profiles that point at `github.com`.
- **Breaking:** profile names, SSH aliases, GitHub logins, Git names, emails,
  key paths, and workspaces are validated; names and aliases must be unique
  ignoring letter case. Invalid values from the command line exit `2`.
- **Breaking:** `fix-remote` and `clone` rewrite only `github.com` URLs and
  other profiles' aliases, always to `git@<alias>:owner/repo.git`
  (`ssh://git@github.com/...` origins are now rewritten to that form too).
- Config writes are atomic (temporary file, flush, rename), keep the file's
  permissions, and preserve unknown keys in order.
- `ghs use` reports the account it switched to and the Git config scope it
  changed (`local: <repository>` or `global`).
- `ghs init-ssh` and `ghs clone` name the key they generated, the SSH config
  block they appended, and (for `clone`) the account they switched to.
- The README states the Go version from `go.mod` (1.26.3), the exit-code
  contract, that only `github.com` is supported, and that `ghs` must not be
  run with `sudo`; a test keeps README and `ghs --help` in sync.

### Fixed

- A config file that could not be read or parsed was treated as empty and
  overwritten, losing every stored profile.
- Config writes truncated the file in place, so an interrupted write could
  leave a partial config.
- SSH `IdentityFile` paths containing spaces were written unquoted; the block
  could also be glued to the previous line when `~/.ssh/config` lacked a
  trailing newline, and an existing `Host` line in a different letter case was
  not detected.
- `init-ssh` generated a key before discovering that `~/.ssh/config` was
  unreadable, and accepted a private key whose `.pub` file was missing.
- `init-ssh --upload` uploaded the key to whichever account happened to be
  active; it now switches to the profile's account and switches back, on
  success and on failure, and treats an already-registered key as success.
- `ghs use` switched the GitHub CLI account and then failed on the Git
  identity (for example outside a repository), leaving the user half
  switched; it now checks everything first and rolls back on failure.
- `fix-remote` rewrote GitLab, GitHub Enterprise, and unknown SSH-host remotes,
  and called `set-url` even when `origin` was already correct.
- `clone` accepted non-GitHub URLs and only failed after switching accounts.
- A mistyped flag such as `--globl` was silently ignored.
- `import-all` could store profiles for an account whose derived alias
  collided with an existing profile.

## [0.4.0] - 2026-06-10

Released before this changelog was kept (historical tag). Added `ghs update`.

## [0.3.0] - 2026-06-10

Released before this changelog was kept (historical tag). Added `ghs version`.

## [0.2.0] - 2026-06-10

Released before this changelog was kept (historical tag). Wrote SSH
`IdentityFile` paths with forward slashes on Windows.

## [0.1.0] - 2026-06-10

Released before this changelog was kept (historical tag). First release.

[Unreleased]: https://github.com/izzamoe/ghs/compare/v0.5.1...HEAD
[0.5.1]: https://github.com/izzamoe/ghs/releases/tag/v0.5.1
[0.5.0]: https://github.com/izzamoe/ghs/releases/tag/v0.5.0
[0.4.0]: https://github.com/izzamoe/ghs/releases/tag/v0.4.0
[0.3.0]: https://github.com/izzamoe/ghs/releases/tag/v0.3.0
[0.2.0]: https://github.com/izzamoe/ghs/releases/tag/v0.2.0
[0.1.0]: https://github.com/izzamoe/ghs/releases/tag/v0.1.0
