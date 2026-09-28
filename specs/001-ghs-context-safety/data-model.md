# Data Model: ghs Account-Context Safety

**Feature**: `001-ghs-context-safety` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

`ghs` has no database. Its state lives in five places, four of which it does not
own. This document names each entity, its fields, its validation rules, and the
states it can be in. Go type names are given because `tasks.md` refers to them;
they live in the packages shown.

## 1. Profile (`config.Profile`, `internal/config/profile.go`)

| Field | Config key | Rule (spec) | Notes |
|-------|-----------|-------------|-------|
| `Name` | section header `[name]` | FR-007: `[A-Za-z0-9._-]`, letters may be Unicode letters, not starting with `-` or `.`, 1–64 chars, not a command name, unique case-insensitively | Case-insensitive uniqueness checked by `Config.CheckUnique` |
| `GitHubUser` | `gh_user` | FR-009: `[A-Za-z0-9]` and single `-`, no leading/trailing `-`, 1–39 chars | May be empty in old files; commands that need it fail naming the field |
| `GitName` | `git_name` | FR-010: non-empty, no control chars, no `"` | |
| `GitEmail` | `git_email` | FR-011: exactly one `@`, non-empty local part, domain with a `.`, no whitespace/control/`"` | May be empty; `use` then switches only the account |
| `SSHHostAlias` | `ssh_host_alias` | FR-008: `[A-Za-z0-9._-]`, not starting with `-`, 1–64, not `github.com`, unique case-insensitively | |
| `SSHKey` | `ssh_key` | FR-012: non-empty, absolute or `~/`-prefixed, no control chars, no `"` | Expanded with `config.ExpandPath` |
| `Workspace` | `workspace` | FR-056: non-empty when set, absolute or `~/`-prefixed, not `~`/`~/`/root, no control chars, no `"`, forward slashes, unique and non-nested across profiles (FR-061) | Optional |
| `Extra` | any other key | FR-005: preserved verbatim, in order | `[]KeyValue{Key, Value}` |

Validation functions (`internal/config/validate.go`): `ValidateName`,
`ValidateLogin`, `ValidateGitName`, `ValidateEmail`, `ValidateKeyPath`,
`ValidateAlias`, `ValidateWorkspace`, and `(Config) CheckUnique(p Profile,
ignoreIndex int) error` for name, alias, and workspace collisions (including
nesting). Every function returns an error that names the field and the rule,
and the CLI wraps it as a `UsageError` when the value came from the command
line (FR-013).

## 2. Config file (`config.Config`, `internal/config/load.go`, `save.go`)

- **Location**: `$XDG_CONFIG_HOME/ghs/config.conf` or `~/.config/ghs/config.conf`.
- **Fields**: `Profiles []Profile` in file order.
- **Load rules** (FR-004): duplicate section names (case-insensitive) → error
  with both line numbers; key before any section → error with line; a value
  starting with `"` must end with `"` → otherwise "unterminated quote" with
  line; any control character in a value → error with line; CRLF accepted.
  Comment lines (`#`) are skipped and not preserved.
- **Save rules** (FR-002, FR-003, FR-006): atomic replace; mode preserved or
  `0600`; directory `0700`; profile order preserved; known keys first in the
  fixed order `gh_user, git_name, git_email, ssh_host_alias, ssh_key,
  workspace`, then `Extra` in original order; empty values omitted.
- **States**: *absent* → *present-valid* → *present-invalid* (parse error) →
  *unreadable* (I/O or permission error). Only *absent* and *present-valid* may
  be written; the other two stop every writing command (FR-001).

## 3. Identity file (`config.IdentityFile`, `internal/config/identity.go`)

- **Location**: `<config dir>/gitconfig-<profile name>`.
- **Content**: exactly

  ```ini
  [user]
  	name = <GitName>
  	email = <GitEmail>
  ```

- **Rules**: written atomically with mode `0600` (FR-054); rewritten whenever a
  linked profile's name or email changes (FR-060); deleted on `--unlink` and
  `remove` (FR-058, FR-064). `ReadIdentityFile` returns name and email for
  `doctor`'s comparison (FR-075).

## 4. Workspace link (`gitops.WorkspaceLink`, `internal/gitops/includeif.go`)

| Field | Derivation |
|-------|-----------|
| `Pattern` | `Workspace` with `\` → `/`, `~/` kept, exactly one trailing `/` |
| `Key` | `includeIf.gitdir/i:<Pattern>.path` |
| `File` | absolute identity file path, forward slashes |

**States** (observed by `doctor`, FR-075; repaired by `workspace`, FR-057):

| State | `workspace` key | `includeIf` entry | identity file | `doctor` outcome |
|-------|-----------------|-------------------|---------------|------------------|
| none | absent | absent | absent | `skip` |
| configured-only | present | absent | absent | `fail` (not linked) |
| file-only | present | absent | present | `fail` (not linked) |
| entry-only | present | present | absent | `fail` (identity file missing) |
| stale | present | present | present, content differs | `fail` (identity differs) |
| linked | present | present | present, content matches | `ok` |

Transitions: `workspace <profile> <path>` moves any state to *linked* (adding
only the missing parts); `--unlink` and `remove` move any state to *none*;
`set-email`, `add-from-gh`, and `import-all` overwrite move *stale* to *linked*
by rewriting the file when it exists (FR-060).

## 5. SSH config block (`sshops.HostBlock`, `internal/sshops/parse.go`)

- **Location**: `~/.ssh/config` (never rewritten; append-only for `init-ssh`
  and `clone`, read-only for `doctor`).
- **Written form** (FR-015–017):

  ```
  Host <alias>
    HostName github.com
    User git
    IdentityFile <path or "quoted path">
    IdentitiesOnly yes
  ```

- **Parsed fields**: `HostName`, `User`, `IdentityFile` (unquoted, `~`
  expanded), `IdentitiesOnly`; `Found bool`.
- **`doctor` rule** (FR-073): all four present with the required values, and
  `IdentityFile` and `IdentityFile + ".pub"` both exist.

## 6. GitHub CLI account set (`ghops.AuthAccount`, `internal/ghops/user.go`)

- **Source**: `gh auth status --hostname github.com --json hosts`.
- **Fields**: `Login`, `Active`, `State` (`success` means healthy), `Host`.
- **Derived**: `ActiveAccount() (login string, ok bool)`;
  `IsAuthenticated(login) bool` (state `success`).
- **Rules**: FR-020 (target must be authenticated before any switch), FR-021–024
  (temporary switch and restore), FR-070 (doctor), FR-077 (list).

## 7. Repository remote (`gitops.Remote`, `internal/gitops/remote.go`)

| Field | Values |
|-------|--------|
| `Kind` | `GitHub` (host `github.com` in scp, `ssh://`, `https://`, `http://` form), `SSHHost` (scp or `ssh://` form with any other host), `Other` (anything else, including non-URL strings) |
| `Host` | lowercased host or scp host token |
| `Path` | `owner/repo.git` normalized: trailing `/` removed, exactly one `.git` |
| `Raw` | the input |

**Resolution against the config** (used by `fix-remote`, `use`, `status`,
`doctor`): `GitHub` → "unaliased github.com"; `SSHHost` with `Host` equal to a
profile alias → that profile; `SSHHost` otherwise → "unknown alias"; `Other` →
"not github". Only "unaliased github.com" and "another profile's alias" are
rewritable (FR-029); "already this profile's alias" is a no-op (FR-031).

## 8. Doctor check (`cli.doctorCheck`, `internal/cli/doctor.go`)

| Field | Values |
|-------|--------|
| `Name` | `gh`, `git identity`, `origin`, `ssh config`, `ssh auth`, `workspace` (fixed order, FR-067) |
| `Outcome` | `ok`, `warn`, `fail`, `skip` |
| `Message` | one line, ending with `; run: <command>` for `fail`/`warn` where a remedy exists |

Exit code: `1` if any `fail`, else `0`. Summary: `summary: <ok> ok, <warn> warn,
<fail> fail, <skip> skip`.

## 9. Mismatch (`cli.mismatch`, `internal/cli/status.go`)

A mismatch is a pair of resolutions that disagree. `status` resolves three
facts to a `resolution{Profile string; Kind string}` where `Kind` is one of
`profile`, `none`, `unaliased`, `unavailable`, `notgithub`, and emits one
`mismatch:` line for each condition in FR-078, in this order: account→none,
identity→none, identity≠account, origin≠account, origin≠identity,
origin unaliased.

## 10. Tool invocation log (tests only, `internal/e2e`)

One JSON object per line in `GHS_FAKE_LOG`:

```json
{"tool":"git","args":["config","--global","--add","includeIf.gitdir/i:~/Documents/work/.path","/tmp/x/config/ghs/gitconfig-work"],"dir":"/tmp/x/repo"}
```

Tests assert on ordered subsequences of `(tool, args)` and on the absence of
mutating operations (see [contracts/fake-tools.md](./contracts/fake-tools.md)
for the mutating-operation list).

## 11. Fake state (tests only)

Declared per test, serialized to `GHS_FAKE_STATE`; schema in
[contracts/fake-tools.md](./contracts/fake-tools.md). Fakes read it on every
invocation and persist the mutations they simulate (account switch, Git config
writes, remote URL changes, key uploads) back to it.
