# Feature Specification: ghs Account-Context Safety

**Feature Branch**: `001-ghs-context-safety`

**Created**: 2026-09-28

**Amended**: 2026-09-28 (scoped features A–E promoted from open questions to
mandatory requirements: `remove`, `doctor`, active-profile and mismatch reporting
in `list`/`status`, `use` origin warning and `--fix-remote`, and `workspace`
implemented as Git conditional-include automation)

**Status**: Draft

**Input**: User description: "Fix ghs account-context safety: config no-data-loss atomic persistence, profile/alias validation, safe SSH config with spaces, correct upload account and restoration, no non-GitHub remote rewrite, no partial use state, reject unsupported GHES/hostname behavior, resolve unused workspace, strict flags, doc/version/release/CI/license, hermetic E2E tests with fake gh/git/ssh."

## Context

`ghs` switches a developer between GitHub identities by changing four things the
GitHub CLI does not change together: the active GitHub CLI account, the Git commit
identity, the SSH host alias in the user's SSH config, and the repository `origin`
remote. An audit of the current release found that several commands can lose the
user's saved profiles, leave the user half-switched between two accounts, upload a
key to the wrong account, rewrite remotes that do not point at GitHub, or silently
ignore mistyped flags. This feature fixes those defects, gives the stored-but-unused
`workspace` setting its intended effect through Git's own conditional includes, adds
`remove` and `doctor`, makes `list`, `status`, and `use` surface account mismatches,
and puts the project on a documented, tested release footing.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Saved profiles are never lost (Priority: P1)

A developer with several profiles runs a profile-writing command (`add-profile`,
`add-from-gh`, `import-all`, `set-email`). Whatever happens (unreadable config file,
malformed line, disk full, interrupted process), the profiles they already had are
still present and intact afterward.

**Why this priority**: losing the config file destroys every stored identity at once
and is the only defect that cannot be undone by re-running a command.

**Independent Test**: seed a config with three profiles, corrupt one line, run each
writing command, and confirm the original file is byte-for-byte unchanged and the
command reported the corruption.

**Acceptance Scenarios**:

1. **Given** a config file with an invalid line, **When** the user runs any command
   that writes a profile, **Then** the command exits with an error naming the line
   and the file is unchanged.
2. **Given** a config file that exists but cannot be read (permissions), **When** the
   user runs `add-profile`, **Then** the command exits with an error and does not
   create, truncate, or replace the file.
3. **Given** a healthy config with profiles A and B, **When** `add-profile C` is run,
   **Then** the resulting file contains A, B, and C, in that order, with all fields of
   A and B unchanged.
4. **Given** a healthy config, **When** the write is interrupted after the file has
   been opened, **Then** the original file is still complete (no partial content).
5. **Given** no config file and no config directory, **When** the user adds a profile,
   **Then** the directory and file are created and readable only by the user.

---

### User Story 2 - Invalid names and values are rejected before they are stored (Priority: P1)

A developer mistypes or pastes a profile name, SSH alias, key path, email, or user
name that would corrupt the config file or the SSH config (spaces, quotes, brackets,
newlines, leading dash). The command refuses with a clear message instead of writing
a file that another tool will misread.

**Why this priority**: a corrupted config or SSH config silently breaks every later
command and is hard for the user to diagnose.

**Independent Test**: run `add-profile` with each malformed value and confirm a usage
error, exit code `2`, and no file change.

**Acceptance Scenarios**:

1. **Given** a profile name containing `]`, whitespace, `/`, or a control character,
   **When** it is passed to `add-profile` or `add-from-gh`, **Then** the command
   rejects it as a usage error and nothing is written.
2. **Given** an SSH alias containing whitespace, `#`, `"`, or a control character,
   **When** it is supplied or generated, **Then** the command rejects it as a usage
   error and nothing is written.
3. **Given** a GitHub login, Git name, or email containing a newline, carriage return,
   or double quote, **When** it is supplied on the command line or returned by the
   GitHub CLI, **Then** the command rejects it and names the offending field.
4. **Given** an email that is not of the form `local@domain` with a non-empty local
   part and a domain containing at least one dot, **When** passed to `set-email`,
   `add-profile`, or `--git-email`, **Then** the command rejects it as a usage error.
5. **Given** an SSH key path that is empty, relative without `~`, or contains a
   newline or double quote, **When** supplied, **Then** the command rejects it.
6. **Given** a profile name that differs from an existing one only by letter case,
   **When** `add-profile` is run, **Then** the command rejects it as a duplicate and
   suggests the existing name.
7. **Given** a profile name whose derived SSH alias would be identical to another
   profile's alias, **When** `add-from-gh` or `import-all` derives that alias, **Then**
   the command rejects the collision and tells the user to pass an explicit alias.

---

### User Story 3 - SSH config is safe with spaces in paths (Priority: P1)

A developer whose home directory or key path contains spaces (common on Windows and
macOS) runs `init-ssh` or `clone`. The SSH config block that `ghs` writes is accepted
by the SSH client as written, and re-running the command never adds a second block.

**Why this priority**: an unquoted path with spaces makes SSH fail on every clone and
push for that profile, and the user has no indication that `ghs` caused it.

**Independent Test**: set a sandbox home containing a space, run `init-ssh`, and
confirm the key path in the written block is quoted and the block is added once.

**Acceptance Scenarios**:

1. **Given** a key path with a space, **When** `init-ssh` writes the block, **Then**
   the identity file value is enclosed in double quotes with forward slashes.
2. **Given** a key path without spaces, **When** the block is written, **Then** it is
   written unquoted, preserving today's format for existing users.
3. **Given** an SSH config already containing a `Host` line that lists the alias
   (alone, or among several patterns, in any letter case), **When** `init-ssh` runs,
   **Then** the file is not modified at all, including its modification time.
4. **Given** an SSH config that does not end with a newline, **When** a block is
   appended, **Then** the new block starts on its own line.
5. **Given** an SSH config that exists but cannot be read, **When** `init-ssh` runs,
   **Then** the command fails before generating a key or writing anything.

---

### User Story 4 - The SSH key is uploaded to the right account and the previous account is restored (Priority: P1)

A developer with accounts `work` and `me` active in the GitHub CLI runs
`ghs init-ssh work --upload` while `me` is the active account. The key is added to
`work`, and `me` is the active account again afterward. `ghs clone` with
`--upload-key` behaves the same way with respect to the upload target.

**Why this priority**: uploading a key to the wrong account grants push access to the
wrong identity and is a security defect, not just an inconvenience.

**Independent Test**: with fake GitHub CLI state where the active account differs from
the profile's account, run `init-ssh --upload` and assert the recorded sequence:
switch to profile account, upload, switch back.

**Acceptance Scenarios**:

1. **Given** the active GitHub CLI account differs from the profile's account,
   **When** `init-ssh <profile> --upload` runs, **Then** the key is uploaded while the
   profile's account is active and the previously active account is active on exit.
2. **Given** the upload fails, **When** `init-ssh --upload` runs, **Then** the previous
   account is restored, and the error output names both the upload failure and the
   restoration result.
3. **Given** the profile's account is not authenticated in the GitHub CLI, **When**
   `init-ssh --upload` runs, **Then** the command fails before generating a key or
   writing SSH config.
4. **Given** the profile's account is already active, **When** `init-ssh --upload`
   runs, **Then** no account switch is performed.
5. **Given** the public key is already registered on the account, **When** upload
   runs, **Then** the command succeeds and reports the key was already present.
6. **Given** `ghs clone <profile> ... --upload-key`, **When** it runs, **Then** the
   upload happens while the profile's account is active, and the profile's account
   remains active afterward, because `clone` is documented as switching to it.

---

### User Story 5 - `ghs use` leaves no partial state (Priority: P1)

A developer runs `ghs use work` and the Git identity update fails (for example, they
are not inside a Git repository and did not pass `--global`). Afterward, the active
GitHub CLI account is what it was before the command ran.

**Why this priority**: a half-applied switch is the exact class of confusion the tool
exists to prevent.

**Independent Test**: run `use` with a fake Git that fails on `config user.email` and
assert that the GitHub CLI account was switched back.

**Acceptance Scenarios**:

1. **Given** the current directory is not inside a Git repository and `--global` is
   not given, **When** `ghs use <profile>` runs, **Then** the command fails before
   switching the GitHub CLI account, with a message suggesting `--global`.
2. **Given** all preconditions hold, **When** the Git identity write fails after the
   account switch, **Then** the previous account is restored and both errors are
   reported.
3. **Given** the profile has no email, **When** `ghs use <profile>` runs, **Then** the
   command switches only the GitHub CLI account and states that Git identity was
   left unchanged and why.
4. **Given** the profile's account is not authenticated in the GitHub CLI, **When**
   `ghs use <profile>` runs, **Then** the command fails before changing anything.
5. **Given** success, **When** the command finishes, **Then** the output names the
   account switched to and the Git config scope (local path or global) that changed.

---

### User Story 6 - Remotes that do not point at GitHub are left alone (Priority: P2)

A developer runs `ghs fix-remote work` inside a repository whose `origin` points at
GitLab, a self-hosted server, or a GitHub Enterprise Server host. The command refuses
to rewrite it and explains why.

**Why this priority**: rewriting a non-GitHub remote breaks push and pull for that
repository and is a silent data-flow change.

**Independent Test**: with fake Git returning a GitLab URL, run `fix-remote` and
confirm the remote is unchanged and the error names the host.

**Acceptance Scenarios**:

1. **Given** `origin` is `git@gitlab.com:group/repo.git`, **When** `fix-remote` runs,
   **Then** the command fails, names the host, and does not change the remote.
2. **Given** `origin` is `git@github-other:owner/repo.git` and `github-other` is the
   SSH alias of another `ghs` profile, **When** `fix-remote work` runs, **Then** the
   remote is rewritten to the `work` alias.
3. **Given** `origin` is `git@some-alias:owner/repo.git` and `some-alias` is not the
   alias of any `ghs` profile, **When** `fix-remote` runs, **Then** the command fails
   and does not change the remote.
4. **Given** `origin` already uses the profile's alias, **When** `fix-remote` runs,
   **Then** the command succeeds, reports "already correct", and does not call the
   remote-update operation.
5. **Given** `origin` is an `https://github.com/...` or `ssh://git@github.com/...`
   URL, **When** `fix-remote` runs, **Then** the remote is rewritten to the alias
   form and the repository path (including `.git` suffix presence) is preserved.
6. **Given** `origin` is `https://github.com/owner/repo/` with a trailing slash or a
   `.git` suffix, **When** `fix-remote` runs, **Then** the result is
   `git@<alias>:owner/repo.git`.
7. **Given** `ghs clone` receives a URL whose host is not `github.com`, **When** it
   runs, **Then** it fails before switching accounts or touching SSH config.

---

### User Story 7 - Unsupported hosts are rejected instead of half-supported (Priority: P2)

A developer with a GitHub Enterprise Server account runs
`ghs import-all --hostname ghe.example.com`. Today profiles are created that point at
`github.com` for SSH and account switching. After this change the command explains that
only `github.com` is supported and makes no changes.

**Why this priority**: silently mixing hosts produces profiles that look valid but
cannot work; rejecting up front is cheaper than supporting GHES partially.

**Independent Test**: run `import-all --hostname ghe.example.com` and confirm a usage
error, exit code `2`, no GitHub CLI calls, and no config change.

**Acceptance Scenarios**:

1. **Given** `--hostname` is anything other than `github.com` (case-insensitive),
   **When** `import-all` runs, **Then** the command fails as a usage error before
   calling the GitHub CLI.
2. **Given** `--hostname github.com` or no `--hostname`, **When** `import-all` runs,
   **Then** behavior is unchanged.
3. **Given** the help text and README, **When** a user reads them, **Then** they state
   that only `github.com` is supported.

---

### User Story 8 - A directory is linked to a profile's Git identity (Priority: P2)

A developer keeps work repositories under `~/Documents/work` and personal ones under
`~/code`. They run `ghs workspace work ~/Documents/work` once. From then on every
repository under that directory commits with the `work` identity, and every repository
outside it keeps whatever identity it had, without running `ghs use` per repository.
The `workspace` config key and the `--workspace` flag on `add-profile` and
`add-from-gh` therefore stay and gain this behavior. `ghs` achieves it with Git's own
conditional include (`includeIf "gitdir/i:<directory>/"`), added and removed only
through `git config --global`, never by editing `~/.gitconfig` by hand.

**Why this priority**: the setting is already documented and stored, and the audit's
alternative (removing it) would take away the one mechanism that prevents
wrong-identity commits in repositories the user never ran `ghs use` in.
Directory-scoped identity is also what `doctor` and `status` need in order to explain
where a Git identity came from.

**Independent Test**: with a fake Git recording invocations, run
`ghs workspace work ~/Documents/work` twice; confirm one identity file is written,
exactly one `git config --global --add includeIf...` call is made on the first run and
none on the second, and that `--unlink` removes exactly what was added.

**Acceptance Scenarios**:

1. **Given** profile `work` with a Git name and email, **When**
   `ghs workspace work ~/Documents/work` runs, **Then** (a) a `ghs`-owned identity
   file `gitconfig-work` is written atomically next to the config file, containing
   the profile's `user.name` and `user.email`; (b) the global Git config gains one
   `includeIf.gitdir/i:~/Documents/work/.path` entry pointing at that file, added
   with `git config --global --add`; (c) the profile's `workspace` key is set; and
   (d) the output names all three changes.
2. **Given** the profile is already linked to the same directory, **When** the command
   runs again, **Then** no second `includeIf` entry is added, the identity file is
   left untouched when its content already matches, and the output says the
   workspace was already linked.
3. **Given** the profile has no email, **When** `ghs workspace` runs, **Then** it fails
   with exit code `1` before writing anything and names `set-email`.
4. **Given** a path that is empty, relative without `~/`, equal to the home directory
   or the filesystem root, or that contains a newline, a control character, or a
   double quote, **When** it is passed as the workspace, **Then** the command rejects
   it as a usage error.
5. **Given** `--workspace <path>` on `add-profile` or `add-from-gh`, **When** the
   command runs, **Then** the profile is saved first and then linked exactly as
   `ghs workspace` would; if linking fails after the save, the command exits `1`,
   reports the failure, and tells the user to rerun `ghs workspace <profile> <path>`.
6. **Given** `ghs workspace work --unlink`, **When** it runs, **Then** the `includeIf`
   entry added for that directory is removed with `git config --global --unset`, the
   identity file is deleted, the `workspace` key is removed from the profile, and
   nothing about SSH or the GitHub CLI changes.
7. **Given** a config file written by an earlier version that already contains a
   `workspace` key, **When** any command loads it, **Then** it loads without warning,
   `list` shows the path, and `doctor` reports the workspace as configured but not
   linked until `ghs workspace <profile> <path>` is run.
8. **Given** the workspace directory does not exist yet, **When** it is linked,
   **Then** the command succeeds and notes that the identity applies once
   repositories exist under it.
9. **Given** a linked profile, **When** `set-email` changes its email, or
   `add-from-gh` or `import-all` overwrites it, **Then** the existing `workspace`
   value is preserved and the identity file is rewritten to match the new name and
   email.
10. **Given** the global Git config contains an `includeIf` entry for the same
    directory that `ghs` did not add (a different path value), **When**
    `ghs workspace` links, **Then** the foreign entry is left untouched and the `ghs`
    entry is added alongside it; `--unlink` removes only the `ghs` entry.
11. **Given** another profile is already linked to the same directory, or to a parent
    or child of it, **When** `ghs workspace` runs, **Then** it fails with exit code
    `1` naming the other profile and changes nothing.

---

### User Story 9 - Mistyped flags and arguments are rejected (Priority: P2)

A developer runs `ghs use work --globl` or `ghs list extra`. The command fails with a
usage message rather than silently doing something other than what was asked.

**Why this priority**: today `--globl` silently sets the identity locally in whatever
directory the user happens to be in.

**Independent Test**: run each command with an unknown flag, a missing flag value, a
duplicate flag, and a surplus positional argument; confirm exit code `2` and no
side effects.

**Acceptance Scenarios**:

1. **Given** an unknown flag on any command, **When** it runs, **Then** the command
   exits with a usage error listing the accepted flags for that command.
2. **Given** a value flag with no value (`--git-email` at end of line, or followed by
   another flag), **When** it runs, **Then** usage error.
3. **Given** a boolean flag given a value (`--global yes`), **When** it runs, **Then**
   usage error, because the value would otherwise be treated as a surplus argument.
4. **Given** the same flag twice, **When** it runs, **Then** usage error.
5. **Given** surplus positional arguments on `list`, `status`, `version`, `update`,
   `use`, `init-ssh`, `fix-remote`, `set-email`, `remove`, `doctor`, or `workspace`,
   **When** it runs, **Then** usage error.
6. **Given** flags placed before positional arguments (`ghs use --global work`),
   **When** it runs, **Then** the command behaves the same as with flags after.
7. **Given** `--` as an argument, **When** it runs, **Then** all following arguments
   are positional.
8. **Given** any usage error, **When** it occurs, **Then** the exit code is `2`, no
   external tool is invoked, and no file is modified.
9. **Given** any non-usage failure, **When** it occurs, **Then** the exit code is `1`.
10. **Given** `ghs <command> --help`, **When** it runs, **Then** command-specific
    usage is printed to stdout with exit code `0`.

---

### User Story 10 - Documentation, versioning, release, CI, and license are correct (Priority: P2)

A new user reads the README, installs the stated Go version, runs `go install`, and
gets a versioned binary whose `ghs version` output matches the release tag. A
contributor opens a pull request and CI proves the suite passes on Linux, macOS, and
Windows. Everyone can see the license under which the tool is offered.

**Why this priority**: the README currently states a Go version that cannot build
the module, there is no license, and `ghs update` promises releases that have no
automated path.

**Independent Test**: a repository check confirms the README Go version equals the
module's Go directive, a `LICENSE` and `CHANGELOG.md` exist, a CI workflow runs the
full suite on three platforms, and a tag workflow produces a release.

**Acceptance Scenarios**:

1. **Given** the README, **When** its install section is read, **Then** the required
   Go version equals the module's Go directive, and CI fails if they differ.
2. **Given** a `LICENSE` file, **When** the repository is viewed, **Then** it contains
   the MIT license text with the copyright holder set to the repository owner.
3. **Given** a `CHANGELOG.md`, **When** this feature ships, **Then** it has an
   `Unreleased` section listing every user-visible change from this spec, including
   the new `remove`, `doctor`, and `workspace` commands, the `use` origin warning and
   `--fix-remote` flag, the new `list`/`status` columns and lines, and the stricter
   flag handling marked as breaking.
4. **Given** a version tag is pushed, **When** CI runs, **Then** a GitHub Release is
   created with notes drawn from the changelog and prebuilt binaries for Linux,
   macOS, and Windows on amd64 and arm64.
5. **Given** a pull request, **When** CI runs, **Then** it checks formatting, vet,
   and the full test suite with the race detector on all three platforms, using the
   Go version from the module file.
6. **Given** the README, **When** commands are compared with `ghs --help`, **Then** the
   two agree on every command and flag, and a test enforces it.
7. **Given** the README, **When** read, **Then** it states the exit code contract,
   the `github.com`-only limitation, and that `ghs` must not be run with `sudo`.

---

### User Story 11 - The whole tool can be tested without the network or real accounts (Priority: P1)

A contributor runs the full test suite on any machine. Tests use fake `gh`, `git`,
`ssh`, and `ssh-keygen` executables placed ahead of the real ones on the search path,
a throwaway home directory, and a throwaway config directory. No test reads or writes
the contributor's real SSH, Git, GitHub CLI, or `ghs` state.

**Why this priority**: every other story's acceptance depends on being able to
observe the exact sequence of external calls and the exact file contents.

**Independent Test**: run the suite with the network disabled and a read-only real
home directory and confirm it passes.

**Acceptance Scenarios**:

1. **Given** the suite runs, **When** any command invokes an external tool, **Then**
   the fake receives it and appends the full argument vector to a log the test can
   read.
2. **Given** a test scenario, **When** it declares fake state (authenticated accounts,
   active account, user details, remote URL, global Git config entries, SSH alias
   logins, failing operations), **Then** the fakes answer from that state without any
   network access.
3. **Given** the suite runs, **When** it finishes, **Then** nothing outside the
   temporary directories has changed.
4. **Given** the suite runs on Windows, **When** the fake tools are invoked, **Then**
   they are found and executed exactly as on Linux and macOS.
5. **Given** a story in this spec, **When** the suite is reviewed, **Then** at least one
   end-to-end test exercises each acceptance scenario that involves an external tool
   or a file write, asserting on the recorded call order.

---

### User Story 12 - `use` warns when `origin` still points at `github.com`, or fixes it (Priority: P1)

A developer runs `ghs use work` inside a repository they cloned with plain
`git clone git@github.com:acme/app.git`. The account and Git identity switch, but the
next `git push` still goes through the default SSH key, which may belong to the other
account. After this change `use` tells them so, and `ghs use work --fix-remote`
rewrites `origin` to the profile's alias in the same run.

**Why this priority**: a silent mismatch between the identity `ghs` just set and the
key SSH will actually use is the most common way a commit lands under the wrong
account.

**Independent Test**: with fake Git returning `git@github.com:acme/app.git` as
`origin`, run `use work` and confirm a warning on stderr and no `set-url` call; run
`use work --fix-remote` and confirm the recorded `set-url` call and the new URL in the
output.

**Acceptance Scenarios**:

1. **Given** the current directory is inside a Git repository whose `origin` host is
   `github.com` in any supported form, **When** `ghs use <profile>` runs without
   `--fix-remote`, **Then** the switch completes, the exit code is `0`, and stderr
   carries one warning naming the URL and both remedies (`--fix-remote` and
   `ghs fix-remote <profile>`).
2. **Given** `origin` uses the SSH alias of a different `ghs` profile, **When**
   `ghs use <profile>` runs, **Then** the warning names that other profile.
3. **Given** `origin` already uses the profile's alias, or there is no `origin`, or
   the directory is not inside a repository and `--global` was given, **When** `use`
   runs, **Then** no warning is printed.
4. **Given** `--fix-remote` and a rewritable `origin`, **When** `use` runs, **Then**
   the `origin` rewrite, the account switch, and the Git identity write all happen,
   and the output includes the old and new URL.
5. **Given** `--fix-remote` and an `origin` that is not rewritable (non-GitHub host,
   unknown alias, or missing), **When** `use` runs, **Then** the command fails with
   exit code `1` before switching the account, writing Git identity, or touching the
   remote.
6. **Given** `--fix-remote` outside a Git repository, **When** `use` runs, **Then** it
   fails before any mutation, even when `--global` is also given.
7. **Given** `--fix-remote` and an `origin` already using the alias, **When** `use`
   runs, **Then** no remote-update call is made and the output says the remote was
   already correct.
8. **Given** the account switch or the identity write fails after `--fix-remote`
   rewrote `origin`, **When** `use` runs, **Then** `origin` is set back to its
   previous URL, the account is restored if it was switched, and every result is
   reported with exit code `1`.

---

### User Story 13 - `remove` deletes a profile without deleting anything unrecoverable (Priority: P2)

A developer no longer uses the `old` account and runs `ghs remove old`. The profile
disappears from `ghs list`. Their SSH keys, their SSH config, their GitHub CLI logins,
and the Git identity of every repository are exactly as before, and the output tells
them what was left in place and how to clean it up by hand if they want to.

**Why this priority**: today the only way to drop a profile is to edit the config file
by hand, which is the file the rest of this feature works hardest to protect. A remove
command that also deleted keys or logged accounts out would turn a typo into lost
access.

**Independent Test**: seed three profiles, SSH keys, an SSH config, and fake GitHub CLI
accounts; run `remove` for the middle profile; confirm the config contains the other
two unchanged and in order, and that the SSH directory, SSH config, and fake GitHub
CLI state are byte-identical to before.

**Acceptance Scenarios**:

1. **Given** profiles A, B, C, **When** `ghs remove B` runs, **Then** the config
   contains A and C, in that order, with every key of A and C (including unknown
   keys) unchanged, and the write is atomic.
2. **Given** the profile does not exist, **When** `remove` runs, **Then** exit code
   `1` naming the profile; if a profile differs only by letter case, the message
   suggests it.
3. **Given** the profile has SSH keys on disk and a block in the SSH config, **When**
   `remove` runs, **Then** neither is touched, and the output prints the key path
   and the alias with the manual cleanup steps.
4. **Given** the profile's account is authenticated (or active) in the GitHub CLI,
   **When** `remove` runs, **Then** no `gh auth logout` or `gh auth switch` is
   invoked and the output notes that the account remains logged in; when the GitHub
   CLI is unavailable, the note is omitted and the command still succeeds.
5. **Given** the profile has a workspace, **When** `remove` runs, **Then** the
   workspace is unlinked exactly as `ghs workspace <profile> --unlink` would do it,
   before the config is rewritten; if unlinking fails, the profile is not removed and
   the error is reported.
6. **Given** the profile is the only one, **When** `remove` runs, **Then** the config
   file is left in place and empty, not deleted, and `list` reports no profiles.
7. **Given** any Git repository, **When** `remove` runs, **Then** no `git config`
   write is invoked other than the workspace `includeIf` removal.
8. **Given** a surplus argument or unknown flag, **When** `remove` runs, **Then**
   usage error with exit code `2` and no change.

---

### User Story 14 - `doctor` explains the whole account context in one run (Priority: P2)

A developer whose push was rejected runs `ghs doctor`. In one screen they see which
GitHub CLI account is active and which profile it belongs to, which Git identity
applies in the current directory and where it comes from, whether `origin` goes
through the right SSH alias, whether the alias is correctly configured in the SSH
config, whether GitHub actually authenticates that alias as the expected login, and
whether a workspace link is in place. Each line says `ok`, `warn`, `fail`, or `skip`,
and the exit code is non-zero when anything failed. Nothing is changed by running it.

**Why this priority**: every other story makes a single command safe; `doctor` is how
a user finds out which of the four moving parts is wrong when the symptom is a
rejected push or a wrong commit author.

**Independent Test**: with fake `gh`, `git`, and `ssh`, seed each mismatch class in
turn (wrong active account, wrong Git email, `origin` on `github.com`, `origin` on
another profile's alias, missing SSH block, wrong `IdentityFile`, SSH authenticating
as another login, unlinked workspace) and confirm the corresponding line is `fail` or
`warn` as specified, the summary line is correct, and the invocation log contains only
read operations.

**Acceptance Scenarios**:

1. **Given** everything matches for profile `work`, **When** `ghs doctor` runs,
   **Then** every check prints `ok` (or `skip` with a reason), the summary counts are
   right, and the exit code is `0`.
2. **Given** no profile argument, **When** `doctor` runs, **Then** the profile is the
   one whose `gh_user` is the active GitHub CLI account; if no profile matches, the
   gh check fails, profile-specific checks are `skip`ped with that reason, and the
   exit code is `1`.
3. **Given** `ghs doctor me`, **When** it runs while `work`'s account is active,
   **Then** the gh check fails naming both logins and every other check is evaluated
   against `me`.
4. **Given** the Git identity in the current directory (or globally, outside a
   repository) has a different email from the profile, **When** `doctor` runs,
   **Then** the identity check fails, showing the value, the file it came from, and
   the scope; a differing name alone is `warn`.
5. **Given** `origin` uses `github.com` directly, **When** `doctor` runs, **Then** the
   origin check is `warn` and names `ghs fix-remote <profile>`; **Given** `origin`
   uses another profile's alias, **Then** it is `fail` naming that profile;
   **Given** `origin` is not a GitHub URL, **Then** it is `skip` naming the host;
   **Given** no repository or no `origin`, **Then** `skip`.
6. **Given** the SSH config lacks a `Host` block for the alias, or the block lacks
   `HostName github.com`, `User git`, or `IdentitiesOnly yes`, or its `IdentityFile`
   does not exist or has no `.pub` sibling, **When** `doctor` runs, **Then** the SSH
   config check fails naming the missing element and `ghs init-ssh <profile>`.
7. **Given** the SSH authentication check runs, **When** the SSH client reports
   `Hi <login>!`, **Then** the check is `ok` if the login equals the profile's
   `gh_user` and `fail` naming both logins otherwise; a `Permission denied` result,
   an unresolvable alias, a timeout, or a host-key verification failure is `fail`
   with the SSH client's message and the interactive command to run once.
8. **Given** `--offline`, **When** `doctor` runs, **Then** the SSH authentication
   check is `skip` and nothing on the network is attempted.
9. **Given** the profile has a workspace, **When** `doctor` runs, **Then** the
   workspace check is `ok` when the `includeIf` entry exists and the identity file
   matches the profile, and `fail` naming `ghs workspace <profile> <path>`
   otherwise; without a workspace, `skip`.
10. **Given** any run of `doctor`, **When** the invocation log and file system are
    inspected afterward, **Then** no file was written and no `gh auth switch`,
    `gh auth logout`, `gh ssh-key add`, `git config` write, `git remote set-url`, or
    `ssh-keygen` call was made.
11. **Given** the GitHub CLI, Git, or the SSH client is not installed, **When**
    `doctor` runs, **Then** the corresponding check fails naming the tool and the
    remaining checks still run.

---

### User Story 15 - `list` and `status` name the active profile and every mismatch (Priority: P2)

A developer runs `ghs list` and sees at a glance which profile is active in the GitHub
CLI and which profiles are not logged in at all. They run `ghs status` inside a
repository and see which profile the active account, the current Git identity, and
the `origin` alias each resolve to, followed by an explicit `mismatch:` line for every
disagreement. Neither command changes anything.

**Why this priority**: today `status` prints three raw values and leaves the
comparison to the user; `list` cannot tell which profile is current at all.

**Independent Test**: with fake GitHub CLI state where `work`'s login is active and
`old`'s login is not authenticated, run `list` and confirm the `*` marker, the
`GH AUTH` column values, and the trailing active line; with fake Git returning a `me`
alias for `origin`, run `status` and confirm the `mismatch:` line names both profiles.

**Acceptance Scenarios**:

1. **Given** profiles `work`, `me`, `old` and GitHub CLI accounts where `work`'s
   login is active and `old`'s login is absent, **When** `ghs list` runs, **Then** the
   table has `PROFILE`, `GH USER`, `GIT EMAIL`, `SSH ALIAS`, `WORKSPACE`, `GH AUTH`
   columns; `work` is marked `*` and `active`; `me` shows `yes`; `old` shows `no`;
   and the last line reads `active gh account: <login> (profile "work")`.
2. **Given** the active GitHub CLI account matches no profile, **When** `list` runs,
   **Then** no row is marked and the last line names the login and says no profile
   matches.
3. **Given** the GitHub CLI is missing or fails, **When** `list` runs, **Then** the
   table still prints with `?` in `GH AUTH`, the last line reads
   `active gh account: unknown (<reason>)`, and the exit code is `0`.
4. **Given** a repository where the active account, the Git identity, and the
   `origin` alias all resolve to `work`, **When** `ghs status` runs, **Then** each of
   the three lines ends with `(profile "work")` and no `mismatch:` line is printed.
5. **Given** `origin` resolves to profile `me` while the account is `work`, **When**
   `status` runs, **Then** a `mismatch:` line names both.
6. **Given** the Git identity email matches no profile, **When** `status` runs,
   **Then** the identity line ends with `(no profile)` and a `mismatch:` line says so.
7. **Given** `origin` uses `github.com` directly, **When** `status` runs, **Then** the
   origin line ends with `(github.com, no alias)` and a `mismatch:` line suggests
   `ghs fix-remote <profile>` for the active profile.
8. **Given** the current directory is outside a repository, **When** `status` runs,
   **Then** the Git identity is reported from the global scope, `origin` is reported
   as unavailable, and the exit code is `0`.
9. **Given** `status` resolves the Git identity, **When** it prints the line, **Then**
   the line includes the scope (`local` or `global`) and, when the value comes from a
   `ghs` identity file, the word `workspace`, so the user knows where it comes from.
10. **Given** any of the above, **When** the invocation log is inspected, **Then**
    only read operations were performed.

---

### Edge Cases

- Config file with duplicate profile sections: the loader rejects the file naming
  both line numbers; no command proceeds.
- Config file with a key before any section, an unterminated quote, or a value
  containing a raw newline: rejected with the line number.
- Config file with a profile missing `gh_user`: loads, but commands needing the
  account (`use`, `clone`, `init-ssh --upload`) fail naming the missing field.
- Config file with unknown keys: preserved through load and save without warning, so
  newer files do not break older binaries. Comment lines are not preserved on save
  (existing behavior, unchanged).
- Profile name equal to a reserved word (`help`, `--help`, `-h`, `version`): rejected
  by name validation because it begins with a dash or matches a command name.
- Profile name with unicode letters: accepted if it contains only letters, digits,
  `.`, `_`, `-`, does not begin with `-` or `.`, and is at most 64 characters.
- SSH alias equal to `github.com`: rejected, because it would shadow the real host.
- SSH alias already used by a different profile: rejected as a collision.
- `init-ssh` when the private key exists but the public key does not: fails and tells
  the user to regenerate the public key; it does not overwrite the private key.
- `init-ssh --upload` when the public key file is unreadable: fails before switching
  accounts.
- `import-all` when the previously active account cannot be restored: succeeds in
  saving profiles, reports the restoration failure, and exits with code `1`.
- `import-all` where two accounts derive the same alias: the second is rejected and
  the command fails before saving, restoring the active account.
- `use --global` inside a repository: only the global identity changes; the `origin`
  warning of Story 12 still applies because the directory is inside a repository.
- `use --fix-remote --global` inside a repository: `origin` is rewritten and the
  global identity is set; outside a repository it fails before any mutation.
- `clone` into an existing non-empty directory: the Git clone fails; the account was
  already switched and stays switched, as documented for `clone`; the SSH block
  written stays, because it is correct and idempotent.
- `fix-remote` when there is no `origin`: fails naming the missing remote.
- `fix-remote` when the profile's alias is not in the SSH config: succeeds in
  rewriting but warns that `init-ssh <profile>` is needed before the remote works.
- `status` outside a Git repository: prints the account section, reports the global
  Git identity, marks `origin` as unavailable, exit code `0`.
- `update` when the binary is a local build: fails with the install command, exit `1`.
- Home directory cannot be resolved: every command fails with exit `1` before any
  external call.
- The GitHub CLI is not installed: commands that need it fail with a message that
  names the tool and the install page; `list`, `status`, `remove`, `set-email`, and
  `version` still work (with the `GH AUTH` column and account notes degraded to
  unknown).
- `workspace` with a path on Windows using backslashes: stored and matched with
  forward slashes (`C:/Users/x/work/`).
- `workspace` when the global Git config cannot be read (`git config --global
  --get-all` fails for a reason other than "not found"): fails before writing.
- `workspace` when the profile's identity file exists but the `includeIf` entry is
  missing (or the reverse): the link is repaired to the full state; the output names
  what was added.
- `remove` while the current directory's `origin` uses the profile's alias: the
  remote is not changed; `status` afterward shows `(no profile)` for `origin`.
- `doctor --offline` on a machine without an SSH client: the SSH auth check is
  `skip` (offline), not `fail`, because the tool is not needed.
- `doctor` when the SSH client times out: `fail` with the client's message.
- `doctor` when `~/.ssh/config` is unreadable: the SSH config check fails naming the
  read error; other checks still run.
- `list` when the GitHub CLI reports accounts for other hosts only: every row shows
  `GH AUTH` `no` and the trailing line reads `active gh account: none for github.com`.

## Requirements *(mandatory)*

### Functional Requirements

**Config persistence**

- **FR-001**: Profile-writing commands MUST load the existing config and stop with an
  error if the file exists but cannot be read or parsed. They MUST NOT treat such a
  file as empty.
- **FR-002**: Config writes MUST be atomic: the complete new content is written to a
  temporary file in the same directory, flushed to disk, and renamed over the target.
  A crash at any point leaves either the old or the new file, never a mix.
- **FR-003**: The config file MUST be created with owner-only read and write
  permissions and its directory with owner-only permissions; existing permissions
  MUST be preserved on rewrite.
- **FR-004**: Loading MUST reject duplicate profile names, keys outside a section,
  unterminated quotes, and values containing control characters, reporting the line
  number.
- **FR-005**: Unknown keys MUST be preserved verbatim across load and save, in their
  original order within the profile, without any warning.
- **FR-006**: Profile order in the file MUST be preserved across load and save; new
  profiles are appended.

**Validation**

- **FR-007**: Profile names MUST match: letters, digits, `.`, `_`, `-`; not beginning
  with `-` or `.`; 1 to 64 characters; not equal to any command name; and unique
  ignoring letter case.
- **FR-008**: SSH aliases MUST match: letters, digits, `.`, `_`, `-`; 1 to 64
  characters; not beginning with `-`; not equal to `github.com`; and unique across
  profiles ignoring letter case.
- **FR-009**: GitHub logins MUST match GitHub's rules: letters, digits, single hyphens,
  not beginning or ending with a hyphen, 1 to 39 characters.
- **FR-010**: Git names MUST be non-empty, contain no control characters, and contain
  no double quote.
- **FR-011**: Emails MUST be non-empty, contain exactly one `@`, a non-empty local
  part, a domain containing at least one `.`, and no whitespace, control characters,
  or double quotes.
- **FR-012**: SSH key paths MUST be non-empty, absolute or beginning with `~/`, and
  contain no control characters or double quotes.
- **FR-013**: Validation MUST apply identically to values from flags, from the GitHub
  CLI, and from the config file, and MUST run before any external tool is invoked or
  any file is written.
- **FR-014**: Derived aliases and key paths (from `add-from-gh` and `import-all`) MUST
  pass the same validation and collision checks as user-supplied ones.

**SSH config**

- **FR-015**: The `IdentityFile` value MUST be written with forward slashes and MUST
  be enclosed in double quotes when it contains whitespace.
- **FR-016**: `ghs` MUST detect an existing `Host` line that names the alias (alone or
  among several patterns, case-insensitively) and MUST NOT modify the file in that
  case.
- **FR-017**: When appending, `ghs` MUST ensure the new block begins on a new line,
  and MUST only ever append; it MUST never rewrite, reorder, or truncate the SSH
  config.
- **FR-018**: `ghs` MUST fail before generating a key if the SSH config exists but
  cannot be read.
- **FR-019**: `ghs` MUST never overwrite an existing private key and MUST fail if the
  private key exists without its public key.

**Account switching and restoration**

- **FR-020**: Commands that switch the active GitHub CLI account MUST first confirm
  the target account is authenticated and healthy for `github.com`; otherwise they
  fail before any mutation.
- **FR-021**: `init-ssh --upload` MUST perform the upload while the profile's account
  is active and MUST restore the previously active account afterward, on success and
  on failure. If the profile's account is already active, no switch is performed.
- **FR-022**: `clone --upload-key` MUST perform the upload while the profile's account
  is active; the profile's account remains active afterward.
- **FR-023**: `import-all` MUST restore the previously active account on every exit
  path, and MUST report a restoration failure with exit code `1` even if profiles
  were saved.
- **FR-024**: When an operation fails after a switch, the error output MUST include
  both the original failure and the restoration result.
- **FR-025**: Uploading a key that is already registered on the account MUST be
  treated as success and reported as already present.

**`use` atomicity**

- **FR-026**: `use` MUST check, before switching accounts, that the profile's account
  is authenticated and that the Git identity target exists (inside a Git repository
  for local scope; the global config is always available).
- **FR-027**: If the Git identity write fails after the account switch, `use` MUST
  switch back to the previously active account and report both results.
- **FR-028**: `use` MUST report what it changed: the account, and the Git config
  scope (repository path or "global").

**Remote rewriting**

- **FR-029**: `fix-remote`, `use --fix-remote`, and `clone` MUST rewrite only URLs
  whose host is `github.com` (SSH scp-style, `ssh://`, `https://`, `http://`, and
  `git@` forms) or whose SSH host is the alias of a `ghs` profile in the config.
- **FR-030**: Any other URL MUST be rejected with an error naming the host, and the
  remote MUST be unchanged.
- **FR-031**: When the remote already uses the profile's alias, `fix-remote` MUST
  succeed without calling the remote-update operation and MUST say so.
- **FR-032**: Rewriting MUST preserve the repository path, normalizing a trailing
  slash away and ensuring exactly one `.git` suffix.

**Hosts**

- **FR-033**: `import-all --hostname` MUST accept only `github.com` (ignoring letter
  case); any other value is a usage error raised before invoking the GitHub CLI.
- **FR-034**: Help text and README MUST state that only `github.com` is supported.

**Workspace flag retention**

- **FR-035**: The `--workspace` flag on `add-profile` and `add-from-gh`, and the
  `workspace` config key, MUST be retained. Their behavior is defined by FR-053 to
  FR-061; no command may store the key without performing the link.

**Strict flags**

- **FR-036**: Every command MUST reject unknown flags, missing flag values, duplicate
  flags, values on boolean flags, and surplus positional arguments as usage errors.
- **FR-037**: Flags MUST be accepted in any position after the command name, and
  `--` MUST end flag parsing.
- **FR-038**: Usage errors MUST exit with code `2`, print the command's usage to
  stderr with the `ghs:` prefix, and MUST occur before any external tool call or
  file write. All other failures MUST exit with code `1`.
- **FR-039**: `ghs <command> --help` MUST print that command's usage to stdout and
  exit `0`.

**Project hygiene**

- **FR-040**: The README's stated Go version MUST equal the module's Go directive,
  enforced by an automated check.
- **FR-041**: The repository MUST contain an MIT `LICENSE` and a `CHANGELOG.md` in
  Keep a Changelog format with an `Unreleased` section covering this feature and
  marking breaking changes.
- **FR-042**: CI MUST run formatting, vet, and the full race-enabled test suite on
  Linux, macOS, and Windows for every pull request and push to `main`, using the Go
  version from the module file.
- **FR-043**: Pushing a `v*` tag MUST produce a GitHub Release with changelog-derived
  notes and binaries for Linux, macOS, and Windows on amd64 and arm64.
- **FR-044**: README and `ghs --help` MUST list the same commands and flags, enforced
  by a test.

**Testing**

- **FR-045**: The test suite MUST include end-to-end tests that execute the built
  `ghs` binary with fake `gh`, `git`, `ssh`, and `ssh-keygen` executables on the
  search path, an isolated home directory, and an isolated config directory.
- **FR-046**: Fakes MUST log every invocation's full argument vector and working
  directory and MUST answer from scenario state declared by the test (accounts,
  active account, user details, emails, registered SSH keys, remote URL, repository
  presence, local and global Git config entries, SSH alias logins, and operations
  that should fail).
- **FR-047**: The suite MUST pass with no network access and MUST NOT read or write
  anything outside test-owned temporary directories.
- **FR-048**: Every acceptance scenario in this spec that involves an external tool
  or a file write MUST have an end-to-end test asserting the recorded call order and
  resulting file contents.

**`use` and the `origin` remote**

- **FR-049**: After a successful switch inside a Git repository, `use` MUST read
  `origin` (read-only) and print exactly one warning line to stderr when the host is
  `github.com` in any form accepted by FR-029, or when the SSH host is the alias of a
  different profile. The warning MUST name the URL (and the other profile, if any)
  and both remedies: `--fix-remote` and `ghs fix-remote <profile>`. The exit code
  stays `0`. No warning is printed when `origin` is absent, already uses the
  profile's alias, or the directory is not inside a repository.
- **FR-050**: `use --fix-remote` MUST compute the rewrite during preflight using
  FR-029 to FR-032. A URL that cannot be rewritten, a missing `origin`, or a working
  directory outside a repository MUST fail with exit code `1` before any mutation,
  regardless of `--global`.
- **FR-051**: With `--fix-remote`, the mutation order MUST be: `origin` rewrite,
  account switch, Git identity write. On failure, earlier mutations MUST be reverted
  in reverse order (`origin` restored to its previous URL, account restored), and
  every result MUST be reported.
- **FR-052**: When `origin` already uses the profile's alias, `--fix-remote` MUST NOT
  call the remote-update operation and MUST say the remote was already correct. The
  output MUST always include an `origin` result line when `--fix-remote` is given.

**Workspace (Git conditional include automation)**

- **FR-053**: The `workspace` value MUST be read and acted on by `workspace`,
  `remove`, `list`, `status`, `doctor`, and every profile-saving command; it MUST
  never be stored as inert data.
- **FR-054**: `workspace <profile> <path>` MUST link by (a) writing a `ghs`-owned
  identity file named `gitconfig-<profile>` in the config directory, atomically per
  FR-002, with mode `0600`, containing only a `[user]` section with `name` and
  `email`; (b) adding one global Git entry
  `includeIf.gitdir/i:<pattern>.path = <identity file path>` via
  `git config --global --add`, only when `git config --global --get-all` of that key
  does not already contain the value; and (c) saving the `workspace` key. The
  pattern MUST be the path with `~/` kept for Git to expand (or absolute otherwise),
  forward slashes only, and exactly one trailing `/`.
- **FR-055**: `ghs` MUST NOT open, parse, or rewrite `~/.gitconfig` or any other Git
  config file directly; every read and write of global Git config MUST go through
  `git config --global`.
- **FR-056**: Workspace paths MUST be non-empty, absolute or beginning with `~/`, not
  equal to the home directory or a filesystem root, and contain no control
  characters or double quotes; violations are usage errors. Paths MUST be stored and
  matched with forward slashes.
- **FR-057**: Linking MUST be idempotent: a repeat run with the same path MUST add no
  entry, MUST leave an up-to-date identity file untouched, and MUST report the
  workspace as already linked. A partially present link (file without entry, or
  entry without file) MUST be completed.
- **FR-058**: `--unlink` MUST remove only the entry whose value equals the `ghs`
  identity file path (fixed-value match), MUST leave every other `includeIf` entry
  untouched, MUST delete the identity file, MUST remove the `workspace` key, and
  MUST succeed when the entry or file is already absent.
- **FR-059**: `workspace` MUST fail with exit code `1` before any mutation when the
  profile has no email, naming `set-email`.
- **FR-060**: Saving a profile that already has a `workspace` (via `set-email`,
  `add-from-gh`, or `import-all` overwrite) MUST preserve the value and, when the
  identity file exists, rewrite it to the new name and email.
- **FR-061**: Two profiles MUST NOT be linked to the same directory or to nested
  directories; the second link attempt MUST fail with exit code `1` naming the other
  profile.

**`remove`**

- **FR-062**: `remove <profile>` MUST delete only that profile's section, atomically
  per FR-002, preserving the order and every key (including unknown keys) of the
  remaining profiles. Lookup is exact; on a miss, exit code `1` with a
  case-insensitive suggestion when one exists.
- **FR-063**: `remove` MUST NOT delete or modify SSH key files, MUST NOT modify the
  SSH config, MUST NOT invoke `gh auth logout` or `gh auth switch`, and MUST NOT
  write Git identity in any scope. The only permitted Git write is the workspace
  `includeIf` removal of FR-058.
- **FR-064**: When the profile has a workspace, `remove` MUST unlink it before
  rewriting the config and MUST abort without touching the config if unlinking fails.
- **FR-065**: `remove` output MUST name the removed profile and list what remains:
  the key path and alias with manual cleanup instructions, and, when the GitHub CLI
  is available, whether the account is still logged in or active. The GitHub CLI
  query is read-only and optional; its failure MUST NOT fail the command.
- **FR-066**: Removing the last profile MUST leave an empty config file with its
  permissions preserved, not a deleted file.

**`doctor`**

- **FR-067**: `doctor [<profile>] [--offline]` MUST run these checks in this order:
  `gh`, `git identity`, `origin`, `ssh config`, `ssh auth`, `workspace`; print one
  line per check beginning with `ok`, `warn`, `fail`, or `skip`; print a final
  `summary:` line with the four counts; and exit `1` when any check is `fail`,
  otherwise `0`.
- **FR-068**: `doctor` MUST be non-mutating: it MUST NOT write any file and MUST NOT
  invoke `gh auth switch`, `gh auth logout`, `gh ssh-key add`, any `git config`
  write, `git remote set-url`, or `ssh-keygen`.
- **FR-069**: Profile selection: the positional argument if given; otherwise the
  profile whose `gh_user` equals the active GitHub CLI account. When neither yields a
  profile, the `gh` check is `fail` and every profile-specific check is `skip` with
  that reason.
- **FR-070**: The `gh` check MUST verify the tool is present, that
  `gh auth status --hostname github.com --json hosts` parses, that an active account
  exists, and that the selected profile's `gh_user` equals the active login; any
  other outcome is `fail` naming the logins involved.
- **FR-071**: The `git identity` check MUST read `user.name` and `user.email` with
  their originating file and scope (`git config --show-origin --show-scope --get`)
  from the current directory, or from the global scope outside a repository; `fail`
  when the email differs from the profile or Git is absent, `warn` when only the name
  differs, `ok` otherwise, always printing value, file, and scope.
- **FR-072**: The `origin` check MUST be `skip` outside a repository or without an
  `origin`; `warn` naming `ghs fix-remote <profile>` when the host is `github.com`;
  `fail` naming the other profile when the SSH host is another profile's alias;
  `skip` naming the host when the URL is not GitHub-related; `ok` when it uses the
  profile's alias.
- **FR-073**: The `ssh config` check MUST read the SSH config without writing, locate
  the `Host` block for the alias, and require `HostName github.com`, `User git`,
  `IdentitiesOnly yes`, and an `IdentityFile` (tilde-expanded, quotes removed) that
  exists with a `.pub` sibling; `fail` names the first missing element and
  `ghs init-ssh <profile>`.
- **FR-074**: The `ssh auth` check MUST run exactly
  `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` and MUST NOT pass
  `StrictHostKeyChecking` or `UserKnownHostsFile` options. An exit status of `1` with
  `Hi <login>!` in the output is `ok` when `<login>` equals the profile's `gh_user`
  and `fail` naming both logins otherwise; `Permission denied`, an unresolvable host,
  a timeout, or `Host key verification failed` is `fail` with the client's message
  and the instruction to run `ssh -T git@<alias>` once interactively. With
  `--offline` the check is `skip`.
- **FR-075**: The `workspace` check MUST be `skip` without a workspace; `ok` when the
  `includeIf` entry is present (read via `git config --global --get-all`) and the
  identity file exists with the profile's name and email; `fail` naming
  `ghs workspace <profile> <path>` otherwise.
- **FR-076**: When a required tool is missing, the affected check MUST `fail` naming
  the tool and the remaining checks MUST still run.

**`list` and `status`**

- **FR-077**: `list` MUST query `gh auth status --hostname github.com --json hosts`
  when the GitHub CLI is available; print the columns `PROFILE`, `GH USER`,
  `GIT EMAIL`, `SSH ALIAS`, `WORKSPACE`, `GH AUTH`; mark the active profile's row
  with `*` and `GH AUTH` `active`; show `yes` or `no` for other profiles; and end
  with `active gh account: <login> (profile "<name>")`, or the no-match, none, or
  unknown variants of Story 15. When the GitHub CLI is missing or fails, `GH AUTH`
  is `?`, the trailing line says `unknown (<reason>)`, and the exit code is `0`.
- **FR-078**: `status` MUST resolve the active account, the Git identity email (with
  scope and, when it comes from a `ghs` identity file, the word `workspace`), and the
  `origin` SSH host each to a profile name or `(no profile)`; print one line per
  fact; and print one `mismatch:` line for each of: account and identity differ,
  account and `origin` differ, identity and `origin` differ, `origin` on
  `github.com` without an alias, identity matching no profile, account matching no
  profile. `status` MUST exit `0` and MUST NOT mutate anything.
- **FR-079**: `list`, `status`, and `remove` MUST work without the GitHub CLI, with
  the account-derived fields degraded to unknown, and exit `0` unless another error
  occurs.

### Key Entities

- **Profile**: a named GitHub identity: GitHub login, Git name, Git email (may be
  empty), SSH alias, SSH key path, optional workspace directory, and any unknown keys
  preserved in order. Names and aliases are unique ignoring case; workspaces are
  unique and non-nested.
- **Config file**: the ordered list of profiles plus any unknown keys, owned by
  `ghs`, written atomically.
- **Identity file**: a `ghs`-owned Git config fragment, `gitconfig-<profile>` in the
  config directory, holding one profile's `user.name` and `user.email`; written
  atomically and referenced only by the workspace `includeIf` entry.
- **Workspace link**: the triple of a profile's `workspace` key, the global Git
  `includeIf` entry derived from it, and the identity file. It is complete when all
  three agree; `doctor` reports any other state.
- **SSH config**: the user's SSH client configuration, not owned by `ghs`, append-only
  from `ghs`'s point of view and read-only for `doctor`.
- **GitHub CLI account set**: the accounts authenticated for `github.com`, one of
  which is active; `ghs` observes and switches this.
- **Repository remote**: the `origin` URL of the current or freshly cloned
  repository; rewritten only when it points at GitHub or a `ghs` alias.
- **Doctor check**: one named check with an outcome of `ok`, `warn`, `fail`, or
  `skip` and a one-line message; six per run in fixed order.
- **Mismatch**: a disagreement between the profiles resolved from the active account,
  the Git identity, and the `origin` alias, or an `origin` that bypasses the alias.
- **Tool invocation log**: in tests, the ordered record of every external command,
  its arguments, and its working directory.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Across 100 randomized runs that interrupt a config write at random
  points, the config file is a complete, parseable file every time.
- **SC-002**: Zero commands can leave the active GitHub CLI account different from
  both the pre-command account and the documented post-command account; every exit
  path is covered by an end-to-end test.
- **SC-003**: 100% of malformed inputs in the validation table (at least 30 cases)
  are rejected with exit code `2` and zero file or tool side effects.
- **SC-004**: The full suite passes on Linux, macOS, and Windows in CI with the
  network disabled, in under five minutes per platform.
- **SC-005**: `ghs --help`, the README command list, and the implemented commands are
  identical, verified by a test that fails on drift.
- **SC-006**: The README Go version and the module's Go directive are equal,
  verified in CI.
- **SC-007**: A tagged release produces downloadable binaries for all six
  platform/architecture pairs within one CI run.
- **SC-008**: Every functional requirement above is traceable to at least one test
  by name in `tasks.md`.
- **SC-009**: For each of the eight seeded mismatch classes in Story 14, `doctor`
  produces the specified `fail` or `warn` line and exit code, and the recorded
  invocation log for every `doctor` run contains only read operations.
- **SC-010**: After `remove`, the SSH directory, the SSH config, and the fake GitHub
  CLI state are byte-identical to before, in every end-to-end `remove` test.
- **SC-011**: Linking a workspace records exactly one `git config --global --add`
  call on the first run and zero on the second; unlinking records exactly one
  `--unset` call and leaves foreign `includeIf` entries in place.
- **SC-012**: `use` without `--fix-remote` in a repository whose `origin` uses
  `github.com` prints exactly one warning line and records zero `set-url` calls.

## Assumptions

- Users authenticate with the GitHub CLI themselves; `ghs` never handles tokens.
- Only `github.com` is in scope; GitHub Enterprise Server support is deferred and
  is rejected explicitly rather than partially handled.
- The license is MIT with the repository owner as copyright holder; the maintainer
  can change this before tagging.
- Tightening flag parsing is an acceptable breaking change because the major version
  is `0`; it is called out in the changelog. `--workspace` is retained and gains
  effect, which is additive for existing config files.
- `ghs update` continues to rely on `go install`; releases exist so that `@latest`
  resolves to a tagged version and prebuilt binaries are available for users who do
  not have Go.
- The existing unit tests for URL parsing, email selection, and config loading
  remain and are extended; end-to-end tests are additive.
- Fake tools are shell-independent executables so that they run on Windows without
  a POSIX shell.
- Git 2.30 or newer is available (for `git config --fixed-value`); `includeIf` with
  `gitdir/i` has been available since Git 2.13.
- The GitHub CLI supports `gh auth status --json hosts` (already required by the
  current release).
- The SSH client is OpenSSH-compatible: `ssh -T git@github.com` succeeds with exit
  status `1` and a `Hi <login>!` greeting, and `BatchMode=yes` prevents prompts.

## Out of Scope

- Supporting GitHub Enterprise Server or per-profile hosts.
- Managing SSH agent state, passphrase-protected keys, or `known_hosts`.
- Rewriting remotes other than `origin`.
- Migrating existing SSH config blocks written by earlier versions.
- Managing `includeIf` entries that `ghs` did not create.
- Deleting SSH keys or logging accounts out of the GitHub CLI; `remove` never does
  this and prints the manual steps instead.
- Repairing mismatches automatically from `doctor`; it only reports and names the
  command that repairs each finding.
