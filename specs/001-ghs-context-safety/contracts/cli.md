# CLI Contract: `ghs`

**Feature**: `001-ghs-context-safety` | **Date**: 2026-09-28 | **Spec**: [spec.md](../spec.md)

This is the user-facing contract that `README.md`, `ghs --help`, and the
end-to-end tests must agree on (FR-044). Output formats are exact: tests assert
on them. `<cfg>` means the config directory, `<home>` the user's home.

## Global rules

| Rule | Value |
|------|-------|
| Exit `0` | success (including `status`, `list` with degraded data, `doctor` with no `fail`) |
| Exit `1` | any runtime failure, restoration failure, or `doctor` with at least one `fail` |
| Exit `2` | usage error: unknown command, unknown flag, missing flag value, duplicate flag, value on a boolean flag, surplus or missing positional, invalid value from the command line |
| stdout | success output only |
| stderr | `ghs: <message>` for errors; `ghs: warning: <message>` for warnings; usage text after a usage error |
| Flags | accepted anywhere after the command name; `--` ends flag parsing; `--help`/`-h` on any command prints that command's usage to stdout and exits `0` |
| Hosts | only `github.com`; any other `--hostname` is a usage error |
| Never | invoke a shell, read/print/upload private key material, run as root (documented) |

## Command list (must match `ghs --help` and README verbatim)

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

## Per-command contract

### `add-profile`

- Positionals: exactly 1. Flags: five required value flags, optional `--workspace`.
- Preflight: validate every value (FR-007–013), load config (fail on
  unreadable/invalid), check uniqueness (name, alias, workspace).
- Mutations: atomic config save; then, if `--workspace`, the workspace link
  (see `workspace`). Link failure after save → exit `1` with
  `ghs: workspace link failed: <reason>; rerun: ghs workspace <name> <path>`.
- stdout: `saved profile "<name>" to <cfg>/config.conf` then workspace lines.

### `add-from-gh`

- Positionals: exactly 1. Optional flags as listed.
- Preflight: config loads; `gh api user` (and `gh api user/emails` when needed);
  values from the GitHub CLI validated identically (FR-013); derived alias and
  key path validated and checked for collisions (FR-014).
- Overwrite of an existing profile of the same name preserves its `workspace`
  and refreshes the identity file if present (FR-060).
- Output as `add-profile`.

### `import-all`

- Positionals: 0. Flags: `--hostname <host>` (must be `github.com`,
  case-insensitive; else usage error before any GitHub CLI call), `--require-email`,
  `--no-overwrite`.
- Mutations: `gh auth switch` per account, atomic config save once, restore of
  the previously active account on every path (FR-023). Restoration failure →
  exit `1` after saving, message names both.
- stdout: `imported <n> profiles from gh host "github.com"`.

### `list`

- Positionals: 0. Flags: none.
- Reads: config; `gh auth status --hostname github.com --json hosts` if `gh` is
  on `PATH`.
- stdout (tab-aligned; first column is `*` for the active profile, otherwise
  blank):

  ```
     PROFILE  GH USER     GIT EMAIL         SSH ALIAS    WORKSPACE         GH AUTH
  *  work     zamyb-work  work@example.com  github-work  ~/Documents/work  active
     me       zamyb       me@example.com    github-me    -                 yes
     old      olduser     (missing)         github-old   -                 no
  active gh account: zamyb-work (profile "work")
  ```

  Trailing line variants: `active gh account: <login> (no profile matches)`,
  `active gh account: none for github.com`,
  `active gh account: unknown (<reason>)` (then `GH AUTH` is `?` for every row).
  With no profiles: `no profiles found` then the trailing line.
- Never mutates. Exit `0` unless the config is invalid.

### `status`

- Positionals: 0. Flags: none.
- Reads: config; `gh auth status ... --json hosts`; `git rev-parse
  --show-toplevel`; `git config --show-origin --show-scope --get user.name` /
  `user.email` (with `--global` outside a repository); `git remote get-url origin`.
- stdout:

  ```
  ghs status
  ----------
  gh account:    zamyb-work (profile "work")
  git identity:  Izzam <work@example.com> (profile "work", local)
  origin:        git@github-me:acme/app.git (profile "me")
  mismatch: origin resolves to profile "me" but gh account resolves to profile "work"
  mismatch: origin resolves to profile "me" but git identity resolves to profile "work"
  ```

  Line variants:
  - `gh account:    unavailable (<reason>)`
  - `gh account:    <login> (no profile)`
  - `git identity:  <name> <<email>> (profile "<p>", global, workspace)` — the
    scope word is `local` or `global`; `workspace` is appended when the origin
    file is a `ghs` identity file
  - `git identity:  <name> <<email>> (no profile, local)`
  - `git identity:  unavailable (<reason>)`
  - `origin:        <url> (github.com, no alias)`
  - `origin:        <url> (unknown alias <host>)`
  - `origin:        <url> (not github: <host>)`
  - `origin:        unavailable (not inside a git repository)` /
    `origin:        unavailable (no origin remote)`

  `mismatch:` lines, in this order, each only when applicable:
  1. `mismatch: gh account resolves to no profile (login <login>)`
  2. `mismatch: git identity resolves to no profile (<email>)`
  3. `mismatch: git identity resolves to profile "<a>" but gh account resolves to profile "<b>"`
  4. `mismatch: origin resolves to profile "<a>" but gh account resolves to profile "<b>"`
  5. `mismatch: origin resolves to profile "<a>" but git identity resolves to profile "<b>"`
  6. `mismatch: origin uses github.com directly; run: ghs fix-remote <profile>` (profile = the account's profile, or `<profile>` literally when none)
- Never mutates. Exit `0`.

### `doctor`

- Positionals: 0 or 1 (`<profile>`). Flags: `--offline`.
- Profile selection: argument, else the profile whose `gh_user` is the active
  account; otherwise the `gh` check fails and profile-specific checks are
  `skip`ped with `no profile selected`.
- Reads only: config; `gh auth status ... --json hosts`; `git rev-parse
  --show-toplevel`; `git config --show-origin --show-scope --get ...`; `git
  remote get-url origin`; `git config --global --get-all includeIf...`;
  `~/.ssh/config`; key files (`Stat` only); the identity file; and
  `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` unless `--offline`.
- stdout (first line, then exactly six check lines, then the summary):

  ```
  ghs doctor: profile "work"
  ok    gh: active account zamyb-work (profile "work")
  ok    git identity: Izzam <work@example.com> (local, /repo/.git/config)
  warn  origin: git@github.com:acme/app.git uses github.com directly; run: ghs fix-remote work
  ok    ssh config: Host github-work -> github.com, IdentityFile /home/u/.ssh/id_ed25519_work
  ok    ssh auth: github authenticated github-work as zamyb-work
  skip  workspace: profile has no workspace
  summary: 4 ok, 1 warn, 0 fail, 1 skip
  ```

  Outcome messages per check (`<p>` = selected profile):

  | Check | Outcome | Message |
  |-------|---------|---------|
  | gh | fail | `gh: gh is not installed; see https://cli.github.com` |
  | gh | fail | `gh: cannot read auth status: <reason>` |
  | gh | fail | `gh: no active account for github.com; run: gh auth login` |
  | gh | fail | `gh: active account <login> matches no profile; run: ghs use <profile>` |
  | gh | fail | `gh: active account <login> is not profile "<p>" (<gh_user>); run: ghs use <p>` |
  | gh | ok | `gh: active account <login> (profile "<p>")` |
  | git identity | fail | `git identity: git is not installed` |
  | git identity | fail | `git identity: email <email> differs from profile <expected> (<scope>, <file>); run: ghs use <p>` |
  | git identity | fail | `git identity: user.email is not set (<scope>); run: ghs use <p>` |
  | git identity | warn | `git identity: name "<name>" differs from profile "<expected>" (<scope>, <file>)` |
  | git identity | ok | `git identity: <name> <<email>> (<scope>, <file>)` |
  | origin | skip | `origin: not inside a git repository` / `origin: no origin remote` / `origin: <url> is not a GitHub remote (host <host>)` |
  | origin | warn | `origin: <url> uses github.com directly; run: ghs fix-remote <p>` |
  | origin | fail | `origin: <url> uses alias of profile "<other>"; run: ghs fix-remote <p>` |
  | origin | fail | `origin: <url> uses unknown alias <host>; run: ghs fix-remote <p>` |
  | origin | ok | `origin: <url> uses alias <alias>` |
  | ssh config | fail | `ssh config: cannot read <path>: <reason>` |
  | ssh config | fail | `ssh config: no Host block for <alias> in <path>; run: ghs init-ssh <p>` |
  | ssh config | fail | `ssh config: Host <alias> is missing <keyword> <value>; run: ghs init-ssh <p>` |
  | ssh config | fail | `ssh config: IdentityFile <path> does not exist; run: ghs init-ssh <p>` / `... has no <path>.pub` |
  | ssh config | ok | `ssh config: Host <alias> -> github.com, IdentityFile <path>` |
  | ssh auth | skip | `ssh auth: --offline` |
  | ssh auth | fail | `ssh auth: ssh is not installed` |
  | ssh auth | fail | `ssh auth: github authenticated <alias> as <login>, expected <gh_user>` |
  | ssh auth | fail | `ssh auth: <first line of ssh output>; run once: ssh -T git@<alias>` |
  | ssh auth | ok | `ssh auth: github authenticated <alias> as <login>` |
  | workspace | skip | `workspace: profile has no workspace` / `workspace: <reason of skipped profile>` |
  | workspace | fail | `workspace: <path> is configured but not linked; run: ghs workspace <p> <path>` |
  | workspace | fail | `workspace: identity file <file> is missing; run: ghs workspace <p> <path>` |
  | workspace | fail | `workspace: identity file <file> differs from profile; run: ghs workspace <p> <path>` |
  | workspace | ok | `workspace: <path> linked via <file>` |

  When no profile is selected, `git identity`, `origin`, `ssh config`,
  `ssh auth`, and `workspace` all print `skip  <name>: no profile selected`.
- Exit `1` if any `fail`, else `0`. Never mutates.

### `set-email`

- Positionals: exactly 2. Flags: none. Email validated (usage error).
- Mutations: atomic config save; identity file rewrite when the profile has a
  workspace and the file exists.
- stdout: `set email for profile "<p>"` then, when applicable,
  `updated identity file <file>`.

### `workspace`

- Forms: `workspace <profile> <path>` or `workspace <profile> --unlink`.
  Both a path and `--unlink` → usage error. Neither → usage error.
- Link preflight: profile exists and has an email (else exit `1`:
  `profile "<p>" has no email; run: ghs set-email <p> <email>`); path valid
  (usage error); no other profile linked to the same or a nested directory
  (exit `1`: `workspace <path> overlaps profile "<other>" (<its path>)`);
  `git` on `PATH`; `git config --global --get-all <key>` succeeds or returns
  "not found".
- Link mutations, in order: identity file (atomic), `git config --global --add`
  when missing, config save with the `workspace` key.
- stdout (one line per change actually made):

  ```
  wrote identity file <cfg>/gitconfig-work
  added git include: includeIf.gitdir/i:~/Documents/work/.path = <cfg>/gitconfig-work
  saved workspace ~/Documents/work for profile "work"
  note: <expanded path> does not exist yet; the identity applies once repositories exist under it
  ```

  Fully linked already: `workspace ~/Documents/work is already linked for profile "work"`.
- Unlink mutations, in order: `git config --global --unset --fixed-value <key> <file>`
  (absent entry is not an error), delete identity file (absent is not an
  error), config save without the key.
- Unlink stdout:

  ```
  removed git include: includeIf.gitdir/i:~/Documents/work/.path = <cfg>/gitconfig-work
  deleted identity file <cfg>/gitconfig-work
  removed workspace from profile "work"
  ```

  Profile without a workspace: `profile "work" has no workspace` (exit `0`).

### `use`

- Positionals: exactly 1. Flags: `--global`, `--fix-remote`.
- Preflight (all before any mutation): config valid, profile exists,
  `gh_user` authenticated (`gh auth status --json hosts`), current active login
  recorded; if not `--global`: `git rev-parse --show-toplevel` succeeds (else
  exit `1`: `not inside a git repository; pass --global to set the global identity`);
  if `--fix-remote`: inside a repository (even with `--global`), `origin`
  exists, and its URL is rewritable (else exit `1` naming the host / missing
  remote / unknown alias).
- Mutations in order: `origin` rewrite (only with `--fix-remote` and only when
  not already correct), `gh auth switch --hostname github.com --user <gh_user>`
  (skipped when already active), `git config [--global] user.name` then
  `user.email` (skipped when the profile has no email).
- Rollback on failure: reverse order; `origin` set back to its previous URL;
  account switched back; every result reported; exit `1`.
- After success (without `--fix-remote`), inside a repository: read `origin`;
  warn on stderr:
  - `ghs: warning: origin <url> still uses github.com; pushes will not use the key of profile "<p>"; run: ghs use <p> --fix-remote  or  ghs fix-remote <p>`
  - `ghs: warning: origin <url> uses the alias of profile "<other>"; run: ghs use <p> --fix-remote  or  ghs fix-remote <p>`
- stdout, in order, each only when it happened:

  ```
  origin updated: git@github.com:acme/app.git -> git@github-work:acme/app.git
  origin already correct: git@github-work:acme/app.git
  switched gh account: zamyb -> zamyb-work
  gh account already active: zamyb-work
  git identity set: Izzam <work@example.com> (local: /path/to/repo)
  git identity set: Izzam <work@example.com> (global)
  git identity unchanged: profile "work" has no email; run: ghs set-email work <email>
  ```

### `clone`

- Positionals: 2 or 3. Flags: `--upload-key`.
- Preflight: profile exists; input URL is `owner/repo` or a `github.com` URL
  (else exit `1` naming the host, before any mutation); directory inferable;
  `gh_user` authenticated; SSH config readable.
- Mutations in order: `gh auth switch` (stays switched, documented), key
  generation if missing, SSH config append if missing, `gh ssh-key add` with
  `--upload-key`, `git clone`, `git -C <dir> config user.name/user.email` when
  the profile has an email.
- stdout unchanged from the current release.

### `init-ssh`

- Positionals: exactly 1. Flags: `--upload`.
- Preflight: profile exists; SSH config readable if present; private key
  without public key → exit `1`; with `--upload`: `gh_user` authenticated and
  active login recorded.
- Mutations: key generation if missing; SSH config append if missing (with
  quoting per FR-015); with `--upload`: switch to `gh_user` if not active,
  `gh ssh-key add <key>.pub --title ghs-<p>`, switch back; "already present"
  is success.
- stdout: `ssh is ready for profile "<p>" via host "<alias>"` and, after upload,
  `uploaded public key <path>.pub to account <gh_user>` or
  `public key <path>.pub already registered on account <gh_user>`.

### `fix-remote`

- Positionals: exactly 1. Flags: none.
- Reads `origin`; classification per FR-029–032; `already correct` path makes
  no `set-url` call.
- stdout: `origin updated: <old> -> <new>` or `origin already correct: <url>`.
  stderr warning when the alias has no SSH config block:
  `ghs: warning: alias <alias> has no block in ~/.ssh/config; run: ghs init-ssh <p>`.

### `remove`

- Positionals: exactly 1. Flags: none.
- Preflight: config valid; profile exists (else exit `1`:
  `profile "<name>" not found` or `profile "<name>" not found; did you mean "<Other>"?`).
- Mutations in order: workspace unlink when present (abort on failure), atomic
  config save without the section. Removing the last profile leaves a
  zero-byte file.
- Reads (optional): `gh auth status --json hosts` for the account note; missing
  or failing `gh` omits the note.
- stdout:

  ```
  removed profile "old" from <cfg>/config.conf
  unlinked workspace ~/Documents/old (removed includeIf entry and <cfg>/gitconfig-old)
  kept: ssh key ~/.ssh/id_ed25519_old (delete by hand if unused)
  kept: ssh config block "Host github-old" in ~/.ssh/config (edit by hand)
  kept: gh account olduser is still logged in; run: gh auth logout --hostname github.com --user olduser
  ```

  The last line reads `... is still active; run: ...` when the account is the
  active one.

### `version`, `update`

- Unchanged behavior; strict argument handling (no positionals, no flags).

## Usage text

Each command's usage block is the single line from the command list above plus
one line per flag: `  --flag <value>   description`. Printed to stdout on
`--help`, to stderr after `ghs: <message>` on a usage error.
