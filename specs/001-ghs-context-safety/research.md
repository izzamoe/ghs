# Research: ghs Account-Context Safety

**Feature**: `001-ghs-context-safety` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

Every decision below was checked against the current source tree
(`internal/cli`, `internal/config`, `internal/ghops`, `internal/gitops`,
`internal/sshops`, `internal/runner`) and, where a tool's behavior matters,
against the locally installed tools (`gh 2.101.0`, `git 2.55.0`,
`OpenSSH 10.5p1`). No unresolved clarification markers remain in
[plan.md](./plan.md).

## R1. Atomic config persistence

- **Decision**: `config.Save` writes to `os.CreateTemp(dir, ".config.conf-*")`,
  `Sync()`s, `Close()`s, applies the target's existing mode (or `0600` when the
  target does not exist), then `os.Rename`s over the target. `Save` takes the
  loaded `Config` (including unknown keys) and never starts from an empty struct.
- **Rationale**: today `Save` opens the target with `O_TRUNC` (`internal/config/save.go:14`),
  so a crash or disk-full mid-write leaves a truncated file, and
  `saveProfiles` in `internal/cli/add_profile.go:120-122` replaces a load error with
  an empty config, which then overwrites every stored profile (FR-001, FR-002).
  `os.Rename` replaces atomically on POSIX and uses `MOVEFILE_REPLACE_EXISTING`
  on Windows, so one implementation covers all three platforms.
- **Alternatives considered**: write-then-copy (not atomic); backup file
  `config.conf.bak` (leaves two sources of truth and still truncates the primary
  on failure); file locking (does not address partial writes).

## R2. Strict, position-independent flag parsing

- **Decision**: replace `parseFlags` / `hasFlag` (`internal/cli/add_profile.go:213`,
  `internal/cli/app.go:119`) with a hand-written parser in `internal/cli/args.go`:
  each command declares a `cmdSpec{positional min/max, flags []flagSpec{name,
  takesValue}}`; the parser returns positionals, a `map[string]string`, and a
  `*UsageError` for unknown flags, missing values, duplicate flags, values on
  boolean flags, and surplus positionals. `--` ends flag parsing. `--help`/`-h`
  on any command prints that command's usage to stdout and exits `0`.
- **Rationale**: the standard `flag` package stops at the first positional
  argument, prints its own usage text to stderr, and calls `os.Exit` by default;
  making it interleave flags and positionals and produce the `ghs:`-prefixed
  usage contract of FR-036–039 costs more than the ~100 lines of a direct parser.
  A direct parser also makes "no external tool call before usage validation"
  (FR-038) trivially true because parsing is pure.
- **Alternatives considered**: `flag.FlagSet` with `ContinueOnError` and a
  re-parse loop for interspersed positionals (fragile with `--`); `spf13/pflag`
  (third-party dependency, forbidden by Principle IV without justification).

## R3. Exit-code mapping

- **Decision**: `cli.UsageError` (a struct implementing `error` and carrying the
  command's usage text) is returned for every usage failure. `cmd/ghs/main.go`
  does `errors.As(err, &usageErr)` and exits `2`; every other error exits `1`.
  `App.PrintError` prints `ghs: <message>` then, for usage errors, the usage block.
- **Rationale**: keeps the exit-code policy in one place and lets end-to-end tests
  assert the code directly (FR-038).
- **Alternatives considered**: sentinel `errors.Is(err, ErrUsage)` (loses the
  per-command usage text); exiting from inside `internal/cli` (untestable).

## R4. Workspace as Git conditional include

- **Decision**: `ghs workspace <profile> <path>` writes
  `<config dir>/gitconfig-<profile>` (atomic, `0600`) with

  ```ini
  [user]
  name = <git_name>
  email = <git_email>
  ```

  and runs `git config --global --add "includeIf.gitdir/i:<pattern>.path" <file>`
  unless `git config --global --get-all "includeIf.gitdir/i:<pattern>.path"`
  already lists `<file>`. `<pattern>` is the workspace path with `~/` preserved
  (Git expands it), backslashes converted to `/`, and exactly one trailing `/`.
  `--unlink` runs `git config --global --unset --fixed-value
  "includeIf.gitdir/i:<pattern>.path" <file>` and deletes the file.
- **Rationale**: `git help config` (2.55) documents that `~/` is substituted with
  `$HOME`, a trailing `/` auto-appends `**`, and symlinked and real paths both
  match. `gitdir/i` is used unconditionally so the link works on the
  case-insensitive default filesystems of macOS and Windows without a platform
  switch; on Linux the only effect is that `~/Work/` and `~/work/` both match,
  which is harmless for a directory the user chose. `git config --global` is the
  mechanism `ghs` already uses for `user.name`/`user.email`, so no new file is
  hand-edited (FR-055). `--fixed-value` (Git 2.30, December 2020) avoids treating
  the identity file path as a regular expression. Keeping the identity in a
  `ghs`-owned file means `set-email` can update it atomically and `remove` can
  delete it without parsing `~/.gitconfig`.
- **Alternatives considered**: removing the flag (the original Story 8; rejected
  by the amended scope because directory-scoped identity is the only protection
  for repositories where `ghs use` was never run); writing `[includeIf]` blocks
  into `~/.gitconfig` directly (violates Principle I's "files not owned by ghs are
  append-only" and would need a parser); per-repository `git config` on every
  clone (does not cover existing repositories); `gitdir:` with a per-OS
  case-sensitivity switch (more code for no user benefit).
- **Nested and duplicate workspaces**: rejected at link time (FR-061) because Git
  applies matching includes in file order, so nested links would silently depend
  on the order in which the user ran `ghs workspace`.

## R5. Non-mutating SSH authentication check in `doctor`

- **Decision**: run `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>`
  with combined stdout/stderr captured, ignoring the exit status for
  classification. Outcomes: output containing `Hi <login>! You've successfully
  authenticated` → `ok` if `<login>` equals `gh_user`, else `fail`; output
  containing `Permission denied` → `fail`; `Host key verification failed` →
  `fail` with "run `ssh -T git@<alias>` once interactively"; `Could not resolve
  hostname` or `Connection timed out` → `fail` with the message; anything else →
  `fail` with the first line of output. `--offline` skips the check.
- **Rationale**: GitHub closes the session with exit status `1` after the
  greeting because it grants no shell, so the exit status alone cannot distinguish
  success from failure. `BatchMode=yes` guarantees no passphrase or host-key
  prompt can block a non-interactive run. Not passing `StrictHostKeyChecking` or
  `UserKnownHostsFile` means `doctor` can never add a host key or bypass the
  user's `known_hosts` (FR-074), which keeps the check non-mutating (FR-068).
- **Alternatives considered**: `gh ssh-key list` (proves the key is on the
  account, not that the alias resolves to it); `ssh -G` (prints the effective
  config but does not authenticate); `-o StrictHostKeyChecking=accept-new`
  (writes `known_hosts`, rejected).

## R6. GitHub CLI interfaces

- **Decision**: keep `gh auth status --hostname github.com --json hosts` as the
  single source of authenticated and active accounts (already used by
  `internal/ghops/user.go:62`); add `ghops.ActiveAccount` and
  `ghops.IsAuthenticated` on top of `ParseAuthAccounts`. Add
  `ghops.WithAccount(login, fn)` which switches only when the target is not
  already active, runs `fn`, and switches back, joining errors. Add
  `ghops.AddSSHKey(pubPath, title)` which treats exit `0` as success and a
  non-zero exit whose stderr contains `already` (GitHub's "key is already in use"
  validation message) as "already present".
- **Rationale**: `gh 2.101.0` documents `--json hosts` and `--active` on
  `auth status`; JSON avoids parsing human-readable output. The already-present
  case matters because `init-ssh --upload` is documented as safe to re-run.
- **Alternatives considered**: `gh api user` to discover the active login
  (extra network call and no view of the other accounts); `gh ssh-key list`
  before add (extra call; the add error is sufficient).

## R7. Git identity provenance

- **Decision**: `status` and `doctor` read identity with
  `git config --show-origin --show-scope --get user.email` (and `user.name`),
  whose output is `<scope>\t<origin>\t<value>` where `<origin>` is
  `file:<path>`. Outside a repository, `--global` is added. A value whose origin
  path equals a `ghs` identity file is reported with the word `workspace`.
- **Rationale**: this is the only way to tell the user where an identity came
  from, and it is exactly what makes a workspace link observable (Story 15
  Scenario 9). Both flags exist since Git 2.26.
- **Alternatives considered**: `git config --list --show-origin` (larger output,
  same information); reading `.git/config` and `~/.gitconfig` directly (violates
  FR-055 and misses includes).

## R8. Remote URL classification

- **Decision**: replace the prefix chain in `gitops.RewriteGitHubURL`
  (`internal/gitops/git.go:66`) with `gitops.ParseRemote(url) (Remote, error)`
  returning `Kind` (`GitHub`, `SSHHost`, `Other`), `Host`, and `Path`
  (normalized: no trailing `/`, exactly one `.git`). The CLI then decides:
  `GitHub` → rewrite; `SSHHost` whose host is a profile alias → rewrite;
  `SSHHost` equal to the target alias → already correct; anything else → reject
  naming the host. `use`, `fix-remote`, `clone`, `status`, and `doctor` all use
  the same classifier.
- **Rationale**: today any `git@<anything>:` URL is rewritten
  (`internal/gitops/git.go:87-91`), which is how a GitLab remote gets broken
  (FR-029/030). One classifier keeps the warning logic in `use`, the `mismatch`
  logic in `status`, and the `origin` check in `doctor` consistent.
- **Alternatives considered**: `net/url` parsing only (does not handle scp-style
  `git@host:path`); regular expressions per form (harder to keep the `.git` and
  trailing-slash normalization consistent).

## R9. SSH config reading for `doctor` and quoting for `init-ssh`

- **Decision**: `sshops.ParseHostBlock(content, alias)` scans lines, matching
  `Host` lines case-insensitively (reusing `hasHostBlock`'s token logic), then
  collects `keyword value` pairs until the next `Host`/`Match` line, accepting
  `=` or whitespace separators, stripping surrounding double quotes from values,
  and expanding a leading `~`. `EnsureConfig` quotes `IdentityFile` when the
  forward-slashed path contains whitespace (FR-015).
- **Rationale**: OpenSSH's own parser accepts these forms; `doctor` only needs
  the four keywords of FR-073. A full `ssh_config` grammar is out of scope.
- **Alternatives considered**: `ssh -G <alias>` to read the effective config
  (requires the client to be installed for a file-only check and resolves
  defaults that hide a missing block).

## R10. Hermetic end-to-end harness

- **Decision**: package `internal/e2e` (test files only). `TestMain`:
  1. If `filepath.Base(os.Args[0])` (minus `.exe`) is `gh`, `git`, `ssh`, or
     `ssh-keygen`, run the corresponding fake and exit; this lets the test
     binary itself act as every fake tool.
  2. Otherwise build `ghs` once with
     `go build -buildvcs=false -o <tmp>/ghs[.exe] github.com/izzamoe/ghs/cmd/ghs`
     and run the tests.

  Each test creates a sandbox: temporary `HOME` (and `USERPROFILE` on Windows),
  `XDG_CONFIG_HOME`, a `bin` directory containing copies (or hard links) of the
  test binary named `gh`, `git`, `ssh`, `ssh-keygen` (`.exe` on Windows), a
  state file (`GHS_FAKE_STATE`, JSON) and a log file (`GHS_FAKE_LOG`, one JSON
  object per line). `ghs` is executed with `cmd.Env` set to exactly
  `PATH=<bin>`, `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GHS_FAKE_STATE`,
  `GHS_FAKE_LOG`, plus `SYSTEMROOT`/`TEMP` on Windows; the test process's own
  `PATH` is never modified, so `go build` never sees the fakes.
- **Rationale**: FR-045–047 require the real binary, fakes ahead of real tools,
  no network, and no access to the developer's state. Reusing the test binary
  as the fake avoids four extra `main` packages and works on Windows without a
  shell. Per-command environment isolation is stronger than `t.Setenv` because
  nothing in the test process inherits it.
- **Alternatives considered**: in-process `cli.New(...).Run(...)` (does not test
  exit-code mapping or subprocess environment handling); shell-script fakes
  (fail FR-046 on Windows); `testscript` (third-party dependency).

## R11. Local quality gates and CI

- **Decision**: a `Makefile` with `fmt` (`gofmt -l` must be empty), `vet`,
  `test` (`go test -race ./...`), `e2e` (`go test -race ./internal/e2e/...`),
  and `check` (all of the above); README/help parity and README/`go.mod`
  version parity are Go tests in `internal/cli/docs_test.go` so they run on all
  platforms without shell tooling. CI: `.github/workflows/ci.yml` with a matrix
  of `ubuntu-latest`, `macos-latest`, `windows-latest`, `actions/setup-go` with
  `go-version-file: go.mod`, steps `gofmt -l`, `go vet ./...`, `go test -race
  ./...`. Release: `.github/workflows/release.yml` on `v*` tags running
  GoReleaser (`.goreleaser.yaml`, six targets, `-buildvcs=false`, `ldflags` not
  needed because `ghs version` reads module build info) with release notes
  extracted from the matching `CHANGELOG.md` section by a small Go program in
  `internal/tools/changelog/main.go` executed via `go run`.
- **Rationale**: Principle V requires exactly these gates; keeping every check
  expressible as `go test` or a Go program means Windows contributors can run
  them. GoReleaser is a build-time action, not a runtime dependency, so
  Principle IV is unaffected.
- **Alternatives considered**: `golangci-lint` (useful but not required by the
  constitution; can be added later); a shell script for changelog extraction
  (not portable to Windows contributors verifying locally).

## R12. `remove` safety model

- **Decision**: `remove` is reversible by construction: it deletes one profile
  section from the config (atomic), unlinks the workspace first when present,
  and touches nothing else. Output lists the SSH key path, the SSH config alias,
  and the GitHub CLI login left in place with the manual command for each.
  No confirmation prompt, no `--force`.
- **Rationale**: a prompt is hostile to scripts and is not a safety mechanism
  when nothing irreversible happens; the safety is that keys, SSH config, and
  logins survive and the profile can be re-added from the printed values.
- **Alternatives considered**: `--purge` to also delete keys (rejected: crosses
  into destroying credentials, out of scope); interactive confirmation
  (rejected as above).

## R13. `list` and `status` without the GitHub CLI

- **Decision**: both commands call `runner.LookPath("gh")` first; if absent or
  if `gh auth status` fails, `GH AUTH` becomes `?`, the account line becomes
  `unknown (<reason>)`, and the exit code stays `0`. `remove`'s account note is
  omitted under the same conditions.
- **Rationale**: the existing edge case list promises `list` works without the
  GitHub CLI; the new columns must not break that (FR-079).

## R14. Windows considerations

- **Decision**: all paths written into SSH config or Git config use
  `filepath.ToSlash`; workspace patterns are stored with forward slashes;
  fake tools are `.exe` copies of the test binary; `HOME` and `USERPROFILE`
  are both set in the sandbox; `os.Rename` is relied on for atomic replace.
- **Rationale**: FR-015, FR-056, Story 11 Scenario 4, and the constitution's
  platform constraint.
