# Feature Specification: ghs Public Discoverability and Contributor Readiness

**Feature Branch**: `docs/public-discoverability` (Spec Kit directory `002-public-discoverability`)

**Created**: 2026-09-29

**Status**: Implemented on `docs/public-discoverability` (maintainer actions M1–M9 in tasks.md pending)

**Input**: User description: "Public discoverability and contributor readiness of
the ghs CLI: a user-oriented README with install paths and package-manager/release
choices, first-run workflow, safe multi-account/SSH troubleshooting and recovery,
full commands/flags/help discoverability, config/data privacy, cross-platform
notes, contributor/release documentation; GitHub community health files
(CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, SUPPORT, issue/PR templates, funding
only if honest); repository description/homepage/topics/social preview
recommendations; and automated tests preventing README/help drift. Do not invent
a project website, security contact, funding account, package manager, or
guarantees."

## Context

`ghs` v0.5.0 is published on GitHub and the Go module proxy, but a stranger who
lands on `github.com/izzamoe/ghs` today sees an empty description, no topics, a
community profile at 28% (README and license only), a release whose notes are
empty, and a README that explains each command well but does not tell a new user
how to choose between `go install` and a release archive, what to do on first
run, what to do when a push is rejected or a switch fails halfway, what `ghs`
stores and sends, or how the tool behaves on Windows. Contributors find a
"Contributing" paragraph but no contribution guide, security policy, support
route, or issue template. The existing tests keep the README command list and Go
version in sync with the binary; nothing keeps flag help, install instructions,
changelog links, or community files honest.

This feature makes the repository legible to three readers: a new user, a user
in trouble, and a first contributor. It adds nothing that does not exist: no
package manager, website, email address, funding link, or support promise is
invented. Where the reader would expect one, the documentation says plainly that
it does not exist.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - A new user installs ghs and completes a first switch from the README alone (Priority: P1)

A developer with two GitHub accounts finds the repository, reads the README, and
within one sitting has `ghs` installed by one of the two real paths (Go toolchain
or release archive), verified, and used to switch one repository to the right
account with the right SSH key, without asking anyone.

**Why this priority**: install and first run are the moment a visitor becomes a
user; every later section is useless if this one fails.

**Independent Test**: on a machine with `gh`, `git`, and OpenSSH but without
`ghs`, follow only the README "Install" and "First run" sections; end with
`ghs doctor` reporting no `fail` line for the chosen profile.

**Acceptance Scenarios**:

1. **Given** a reader with Go installed, **When** they read "Install", **Then**
   they see the exact `go install` command, the minimum Go version (equal to the
   `go` directive in `go.mod`), the `PATH` note, and how to verify the install.
2. **Given** a reader without Go, **When** they read "Install", **Then** they see
   how to pick the release archive for their OS and architecture by its exact
   file name pattern, how to verify it against `checksums.txt` on Linux, macOS,
   and Windows, and where to put the binary.
3. **Given** a reader who expects `brew install`, `scoop`, `winget`, `apt`, or an
   AUR package, **When** they read "Install", **Then** the README states that no
   package-manager distribution exists and names the two supported paths.
4. **Given** the README, **When** the reader looks for prerequisites, **Then**
   `gh`, `git` (with minimum version), and an OpenSSH client are listed with what
   each is used for, and "do not run with `sudo`" is stated with the reason.
5. **Given** a reader who installed `ghs`, **When** they read "First run",
   **Then** they see an ordered workflow: log in each account with `gh auth login`,
   import profiles, create and upload SSH keys, switch one repository with
   `--fix-remote`, and confirm with `doctor`, with the expected output of each step.
6. **Given** a reader who wants to remove `ghs`, **When** they read "Uninstall",
   **Then** they see which files to delete and which files `ghs` never deletes for
   them (SSH keys, SSH config blocks, Git config).

---

### User Story 2 - Every command and flag is discoverable from help and README, and they agree (Priority: P1)

A user types `ghs --help`, `ghs <command> --help`, or `ghs help <command>` and
sees the same usage and flag descriptions that the README shows; the README
contains a complete per-command reference, so a user can learn every flag
without running the binary.

**Why this priority**: the constitution's CLI contract (Principle III) requires
help and README to agree; today the README lists usage lines but not the flag
help text, and `ghs help use` silently prints the general help.

**Independent Test**: for every registered command, the README contains the
exact text printed by `ghs <command> --help`; `ghs help <command>` prints that
same text; a test fails when any of them differ.

**Acceptance Scenarios**:

1. **Given** the README, **When** a reader opens the command reference,
   **Then** every command has its usage line(s) and every flag with the same
   description as the binary prints.
2. **Given** the binary, **When** a user runs `ghs help use`, **Then** the output
   equals `ghs use --help` and the exit code is `0`.
3. **Given** the binary, **When** a user runs `ghs help nosuch`, **Then** the
   result is the usage error for an unknown command with exit code `2`.
4. **Given** `ghs --help`, **When** a user reads its last lines, **Then** they
   see where the full documentation lives (the repository README URL), the config
   path, the github.com-only rule, the no-`sudo` rule, and the exit codes.
5. **Given** a change to any flag's help text in the command table, **When** the
   test suite runs, **Then** a test fails until the README is updated.

---

### User Story 3 - A user in trouble recovers using the troubleshooting section (Priority: P1)

A user whose push was rejected, whose commit carries the wrong email, whose
`ghs use` stopped halfway, or whose config file no longer parses finds their
symptom in the README, learns why it happened, and gets both the `ghs` command
that fixes it and the manual `gh`/`git`/editor steps that undo each mutation
`ghs` could have made.

**Why this priority**: `ghs` mutates credentials-adjacent state; a user who
cannot recover by hand loses trust permanently (constitution Principles I and VI).

**Independent Test**: for each symptom row in the troubleshooting table, follow
only the row's instructions in a sandbox with fake tools and reach the stated
end state; every mutation named in the "What ghs changes and how to undo it"
table can be reverted with the listed manual command.

**Acceptance Scenarios**:

1. **Given** `Permission denied (publickey)` on push, **When** the user reads
   the table, **Then** they are told to run `ghs doctor <profile>`, what the
   `ssh config` and `ssh auth` checks mean, and that `ghs init-ssh <profile>
   --upload` creates and registers the key without touching other keys.
2. **Given** `Host key verification failed` from `doctor`, **When** the user
   reads the table, **Then** they learn that `doctor` never writes `known_hosts`
   and that running `ssh -T git@<alias>` once interactively accepts the host key.
3. **Given** a push that used the wrong account, **When** the user reads the
   table, **Then** they are told to run `ghs status` to see which of the three
   resolutions disagrees and `ghs fix-remote <profile>` or `ghs use <profile>` to
   correct it.
4. **Given** a commit with the wrong author email, **When** the user reads the
   table, **Then** they are told to run `ghs use <profile>` (or link a workspace)
   and then `git commit --amend --reset-author` for the last commit.
5. **Given** `ghs use --fix-remote` that failed after rewriting `origin` and
   switching the account, **When** the user reads the output and the table,
   **Then** they see that `ghs` already restored both, what the lines
   `restored origin to ...` and `restored gh account ...` mean, and, if a
   restore line reports failure, the manual `git remote set-url origin <old>`
   and `gh auth switch --user <login>` commands.
6. **Given** `ghs: invalid config <path>: line 7: ...` parse error, **When** the user reads the
   table, **Then** they learn that `ghs` never overwrites an unreadable config,
   where the file is, and that fixing the named line or moving the file aside
   restores operation.
7. **Given** a user who ran `ghs` with `sudo`, **When** they read the table,
   **Then** they learn which root-owned files may now exist and how to remove
   them.
8. **Given** the "What ghs changes" table, **When** the user reads any row,
   **Then** it names the command, the file/account/remote it changes, whether the
   change is atomic, append-only, or via `git config`, and the manual undo.
9. **Given** a Windows user, **When** they read the SSH rows, **Then** they see
   the Windows paths (`%USERPROFILE%\.ssh\config`), that OpenSSH must be
   installed, and that `IdentityFile` is written with forward slashes and quoted
   when the path contains spaces.

---

### User Story 4 - A user understands what ghs stores, reads, and sends, and how it behaves on their platform (Priority: P2)

Before trusting `ghs`, a user reads one section that lists every file it
creates or edits (with permissions), every file it reads, every network request
it causes (through `gh` and `ssh`), and states that it never reads private keys,
never stores tokens, and sends no telemetry. A platform section explains the
Linux, macOS, and Windows differences.

**Why this priority**: identity-switching tools attract scrutiny; explicit
privacy and platform notes prevent wrong assumptions and support requests.

**Independent Test**: every file path, permission, and external command in the
"Config and data" section can be found in the source or the contract in
`specs/001-ghs-context-safety/contracts/cli.md`; nothing is claimed that the
binary does not do.

**Acceptance Scenarios**:

1. **Given** the README, **When** the reader opens "Config and data", **Then**
   they see the config path (identical to the `Config:` line in `ghs --help`),
   its mode `0600`, the identity files `gitconfig-<profile>`, the SSH config
   block format, and that unknown config keys are preserved.
2. **Given** the same section, **When** the reader looks for network activity,
   **Then** every request is listed with its trigger: `gh api user` and
   `gh api user/emails` (add-from-gh, import-all), `gh ssh-key add` with the
   `.pub` file only (init-ssh --upload, clone --upload-key), `ssh -T` (doctor),
   `git clone` (clone), `go install` (update). No other network use exists.
3. **Given** the same section, **When** the reader looks for secrets, **Then**
   they learn that `ghs` stores no tokens (the GitHub CLI holds them), never
   reads or prints private key material, and has no telemetry or update check.
4. **Given** "Cross-platform notes", **When** a Windows reader looks for paths,
   **Then** they see the config, SSH config, and key locations under
   `%USERPROFILE%`, that `XDG_CONFIG_HOME` is honored when set, that backslash
   paths are accepted and normalized, and that files are written with LF endings.
5. **Given** "Cross-platform notes", **When** a macOS reader looks, **Then**
   they see that Apple's bundled OpenSSH and `git` work, that keys are not added
   to the keychain by `ghs`, and that Apple Silicon and Intel archives both exist.

---

### User Story 5 - A first contributor and the maintainer find complete contribution, security, support, and release documentation (Priority: P2)

A contributor opens the repository and finds `CONTRIBUTING.md` (build, gates,
test suite, Spec Kit workflow, pull request expectations), `SECURITY.md`
(scope, how to report privately, what is and is not covered), `SUPPORT.md`
(where to ask, what to include), `CODE_OF_CONDUCT.md`, an issue form for bugs,
one for feature requests, and a pull request template. The maintainer finds a
release runbook and a repository-settings runbook in `CONTRIBUTING.md`.

**Why this priority**: GitHub surfaces these files in the community profile and
in the "new issue" and "new pull request" flows; their absence is the loudest
signal that a project is not ready for outside contributors.

**Independent Test**: `gh api repos/izzamoe/ghs/community/profile` reports every
file present after the change is merged; each file's required content is
asserted by a test.

**Acceptance Scenarios**:

1. **Given** `CONTRIBUTING.md`, **When** a contributor reads it, **Then** they
   find the exact commands to build and run the gates (`make check`, and the
   `gofmt`/`go vet`/`go test -race` equivalents without `make`), the hermetic
   test guarantee, where end-to-end tests live, the test-first rule, the Spec Kit
   sequence for non-trivial changes, and the changelog rule.
2. **Given** `CONTRIBUTING.md`, **When** the maintainer reads the release
   runbook, **Then** they find the steps to cut a release (changelog section,
   link reference, tag on `main`, what the workflow does, how to verify the
   release assets and notes) and the exact commands for every admin-only
   repository setting, each marked as maintainer-only.
3. **Given** `SECURITY.md`, **When** a reporter reads it, **Then** they find the
   supported version policy (latest release only), the private reporting route
   through GitHub's private vulnerability reporting for this repository, the
   fallback when that form is unavailable, what counts as in scope (identity,
   key, config, remote handling), and that there is no bug bounty. No email
   address appears.
4. **Given** `SUPPORT.md`, **When** a user reads it, **Then** they are told to
   run `ghs doctor --offline` and `ghs status`, read the README troubleshooting
   section, search issues, then open a question issue; they are told support is
   best-effort by one maintainer with no response-time promise.
5. **Given** the bug issue form, **When** a user opens it, **Then** it asks for
   `ghs version`, OS and architecture, `gh --version`, `git --version`, the
   command run, expected and actual output, and `ghs doctor --offline` output,
   and reminds them not to paste keys or tokens.
6. **Given** the pull request template, **When** a contributor opens a PR,
   **Then** the checklist asks for `make check`, a `CHANGELOG.md` "Unreleased"
   entry for user-visible changes, README and help updates for CLI changes, and
   a test written first for behavior changes.
7. **Given** the repository, **When** a reader looks for a funding link,
   **Then** none exists, because no funding account exists.

---

### User Story 6 - A visitor understands the project from GitHub metadata before opening the README (Priority: P3)

A visitor who sees the repository in search results, in a social card, or on
the repository page reads a one-line description, sees relevant topics, a
homepage link that resolves, and a social preview that names the tool.

**Why this priority**: metadata drives search and sharing; it is cheap but
cannot be changed by a pull request, so it must be documented as maintainer
actions with exact commands.

**Independent Test**: after the maintainer applies the recorded commands,
`gh repo view --json description,homepageUrl,repositoryTopics,usesCustomOpenGraphImage`
shows the recommended values.

**Acceptance Scenarios**:

1. **Given** the runbook, **When** the maintainer reads it, **Then** each
   metadata item is classified as "changed by files in a PR", "changed by API or
   CLI with admin rights (maintainer confirmation required)", or "web UI only",
   with the exact command or UI path and a read-only verification command.
2. **Given** the description recommendation, **When** applied, **Then** it is
   at most one sentence, names the four things `ghs` switches, and states
   github.com only.
3. **Given** the homepage recommendation, **When** applied, **Then** it points at
   a URL verified to resolve today (the package documentation page on pkg.go.dev)
   or is left empty; no project website is invented.
4. **Given** the topics recommendation, **When** applied, **Then** all topics
   are lowercase, hyphenated, and at most twenty.
5. **Given** the social preview recommendation, **When** applied, **Then** the
   maintainer uploaded an image through the web UI (there is no API), sized
   1280×640, containing only the tool name, the one-line description, and the
   repository path, generated from a source file kept in the repository.

---

### User Story 7 - Documentation cannot silently drift from the code (Priority: P2)

A maintainer changes a flag, a make target, a release asset name, the module
path, the Go version, a changelog heading, or deletes a community file; the
test suite fails and names the stale document.

**Why this priority**: the README is trusted only while it is true; the
constitution (Principle VI) requires mechanical checks for every checkable claim.

**Independent Test**: each drift class below has a test that fails when the
corresponding source and document disagree, and passes on the delivered tree.

**Acceptance Scenarios**:

1. **Given** the command table, **When** any usage line or flag help changes,
   **Then** the README per-command reference test fails.
2. **Given** `go.mod`, **When** the module path or Go version changes, **Then**
   the README install test fails.
3. **Given** `.goreleaser.yaml`, **When** an OS/architecture pair, the archive
   name template, or the checksums file name changes, **Then** the README
   release-archive test fails.
4. **Given** the `Makefile`, **When** a target named in README or CONTRIBUTING
   is removed or renamed, **Then** the make-target test fails.
5. **Given** `CHANGELOG.md`, **When** a `## [x.y.z]` section lacks its link
   reference or `[Unreleased]` compares from an older tag than the newest
   section, **Then** the changelog link test fails (it fails on today's file).
6. **Given** any Markdown file in the documentation set, **When** a relative
   link points at a missing file or an absolute link uses a host outside the
   allowlist, **Then** the link test fails.
7. **Given** the community file set, **When** any file is missing or lacks its
   required content, **Then** the community file test fails.
8. **Given** the six `doctor` check names, **When** the README troubleshooting
   section stops mentioning one, **Then** the troubleshooting coverage test fails.
9. **Given** `ghs --help`, **When** its config path or documentation URL differs
   from the README, **Then** a test fails.

---

### Edge Cases

- README examples use the maintainer's real login and name; the rewrite uses
  neutral placeholders (`alice`, `alice-work`) so readers do not copy real data.
- The Spec Kit branch is `docs/public-discoverability`, which does not match the
  `NNN-name` pattern; Spec Kit scripts are run with
  `SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability` (the installed scripts resolve the
  feature directory from that variable or `.specify/feature.json`, not from
  `SPECIFY_FEATURE`).
- GitHub's private vulnerability reporting is currently disabled; `SECURITY.md`
  must not point at a form that returns 404. The maintainer enables it before the
  file merges, and the file also gives a fallback that does not require it.
- Release v0.5.0 has empty notes although the changelog extractor produces the
  section locally; fixing the published notes is a maintainer action
  (`gh release edit`), not a code change.
- The changelog has a `0.5.1` section but no tag; whether to tag `v0.5.1` is a
  maintainer decision recorded as an optional action, not assumed.
- The homepage may legitimately stay empty; pkg.go.dev is offered because it was
  verified to serve the module today, not because a website exists.
- Contributor Covenant requires an enforcement contact; without an email, the
  contact is the maintainer's GitHub handle and GitHub's private reporting form.
- Windows readers may lack `make`; every `make` target is given with its plain
  command equivalent.
- The README may be read inside a release archive (it is bundled); relative
  links to community files are therefore written so they still identify the file
  by name even when the target is absent from the archive.
- A README rendered on pkg.go.dev shows the same Markdown; badges and relative
  links must not break that rendering (relative links resolve against the module
  root there).

## Requirements *(mandatory)*

### Functional Requirements

**README: install and first run**

- **FR-001**: The README MUST present exactly two install paths, "Go toolchain"
  (`go install <module path>/cmd/ghs@latest`, minimum Go version equal to the
  `go` directive in `go.mod`) and "Release archive", and MUST state that no
  package-manager distribution exists.
- **FR-002**: The release-archive path MUST name the asset pattern
  `ghs_<version>_<os>_<arch>.tar.gz` (`.zip` on Windows) and `checksums.txt`
  exactly as configured in `.goreleaser.yaml`, list every OS/architecture pair
  the release builds, and give checksum verification commands for Linux
  (`sha256sum --check --ignore-missing`), macOS (`shasum -a 256`), and Windows
  PowerShell (`Get-FileHash`).
- **FR-003**: The README MUST list prerequisites with purpose and minimum
  version where one exists: GitHub CLI, Git 2.30 or newer, OpenSSH client; and
  MUST state that `ghs` must not be run with `sudo`, with the reason.
- **FR-004**: The README MUST contain a "First run" section as an ordered
  workflow: `gh auth login` per account, `ghs import-all`, `ghs list`,
  `ghs init-ssh <profile> --upload` per profile, `ghs use <profile> --fix-remote`
  in one repository, `ghs doctor`, with representative output for each step.
- **FR-005**: The README MUST contain "Updating" (`ghs update` for Go installs,
  re-download for archive installs) and "Uninstall" (binary, config directory,
  identity files; and the list of things `ghs` never deletes).

**README: reference and help**

- **FR-006**: The README MUST contain a per-command reference in which each
  command's block equals the output of `ghs <command> --help` byte for byte
  (usage lines and flag help), in command-table order.
- **FR-007**: The README "Commands" summary block MUST continue to equal the
  command list printed by `ghs --help` (existing test retained).
- **FR-008**: `ghs help <command>` MUST print exactly what `ghs <command> --help`
  prints and exit `0`; `ghs help <unknown>` MUST be the unknown-command usage
  error with exit `2`; `ghs help` alone MUST keep printing the general help.
- **FR-009**: `ghs --help` MUST end with a `Docs:` line giving the repository
  README URL (`https://github.com/izzamoe/ghs#readme`), and the README MUST
  contain the same config path text as the help `Config:` line.
- **FR-010**: The README MUST keep the exit-code table and the global flag
  conventions (position-independent flags, `--`, strict rejection) and MUST state
  the stdout/stderr and `ghs:` prefix contract.

**README: troubleshooting and recovery**

- **FR-011**: The README MUST contain a troubleshooting table with at least the
  rows: push rejected `Permission denied (publickey)`; `Host key verification
  failed`; push used the wrong account; commit has the wrong email; `ghs use`
  reported a rollback; rollback reported a failure; config file parse error;
  profile not found / "did you mean"; profile has no email; `gh account ... is
  not logged in`; `origin ... is not a GitHub remote`; `update` refused on a
  local build; ran with `sudo`; Windows OpenSSH missing. Each row MUST give
  cause, the `ghs` command that fixes it, and the manual equivalent.
- **FR-012**: The README MUST contain a "What ghs changes and how to undo it"
  table with one row per mutation: config file (atomic), identity file
  `gitconfig-<profile>` (atomic), `~/.ssh/config` block (append-only), SSH key
  pair (never overwritten, never deleted), global `includeIf` entry (via
  `git config --global`), local/global `user.name`/`user.email`, `origin` URL,
  active GitHub CLI account, public key on the GitHub account, cloned
  directory. Each row MUST name the commands that make the change and the manual
  undo using only `gh`, `git`, or an editor.
- **FR-013**: The troubleshooting section MUST mention each of the six `doctor`
  check names (`gh`, `git identity`, `origin`, `ssh config`, `ssh auth`,
  `workspace`) and explain the four outcomes `ok`, `warn`, `fail`, `skip`.
- **FR-014**: Every recovery instruction MUST be executable without `ghs`
  (Principle VI), and MUST NOT instruct the user to delete SSH keys or log out of
  GitHub CLI accounts.

**README: config, data, and platforms**

- **FR-015**: The README MUST contain a "Config and data" section listing every
  file `ghs` creates or edits with its mode and write strategy, every file it
  reads, every subprocess that reaches the network with the command that
  triggers it, and the statements: no tokens stored, no private key material
  read or printed, no telemetry, no automatic update check.
- **FR-016**: The README MUST document the config file format with one complete
  example profile section and the note that unknown keys are preserved and that
  the file is safe to edit by hand while `ghs` is not running.
- **FR-017**: The README MUST contain "Cross-platform notes" covering Linux
  (`XDG_CONFIG_HOME`), macOS (bundled tools, no keychain integration, both
  architectures), and Windows (`%USERPROFILE%` paths, OpenSSH requirement,
  backslash normalization, forward-slash `IdentityFile`, quoting, LF endings,
  `.zip` archive, PowerShell checksum command).
- **FR-018**: README examples MUST use neutral placeholder identities, not the
  maintainer's real login or name.

**Community health files**

- **FR-019**: `CONTRIBUTING.md` MUST exist and contain: prerequisites, build
  command, `make check` and its non-`make` equivalent, `make e2e`, the hermetic
  test guarantee, test-first rule, Spec Kit sequence and the
  `SPECIFY_FEATURE_DIRECTORY` note, changelog rule, commit/PR expectations, and a
  "Maintainer runbook" with release steps and repository-settings commands
  marked maintainer-only.
- **FR-020**: `SECURITY.md` MUST exist and contain: supported versions (latest
  release only), private reporting through GitHub's private vulnerability
  reporting for `izzamoe/ghs`, a fallback if that form is unavailable (open an
  issue with only the words "security report, please contact me" and no
  details), scope (identity, keys, config, remotes, subprocess argument
  handling), out of scope (vulnerabilities in `gh`, `git`, OpenSSH), the
  statement that there is no bug bounty, and no email address.
- **FR-021**: `CODE_OF_CONDUCT.md` MUST exist, be the Contributor Covenant
  (version 2.1) text with attribution, and name the enforcement contact as the
  maintainer's GitHub handle plus GitHub's private reporting form; no email.
- **FR-022**: `SUPPORT.md` MUST exist and contain the self-help order (`ghs
  doctor --offline`, `ghs status`, README troubleshooting, search issues), the
  issue route, what to include, and the best-effort, single-maintainer,
  no-response-time-promise statement.
- **FR-023**: `.github/ISSUE_TEMPLATE/bug_report.yml`,
  `.github/ISSUE_TEMPLATE/feature_request.yml`, and
  `.github/ISSUE_TEMPLATE/config.yml` MUST exist as GitHub issue forms; the bug
  form MUST require version, platform, command, expected, actual, and
  `ghs doctor --offline` output, and MUST warn against pasting keys or tokens;
  `config.yml` MUST disable blank issues and link to `SUPPORT.md` and the README
  troubleshooting section.
- **FR-024**: `.github/PULL_REQUEST_TEMPLATE.md` MUST exist with the checklist:
  `make check` passed locally, test written first for behavior changes,
  `CHANGELOG.md` Unreleased updated for user-visible changes, README and help
  updated for CLI changes, no new runtime dependency.
- **FR-025**: `.github/dependabot.yml` MUST exist for the `github-actions`
  ecosystem (weekly); no `FUNDING.yml` MUST be created.
- **FR-026**: The README MUST link `CONTRIBUTING.md`, `SECURITY.md`,
  `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, and `LICENSE` by relative
  path.

**Repository metadata**

- **FR-027**: `CONTRIBUTING.md`'s maintainer runbook MUST classify every
  metadata item as PR-changeable, admin API/CLI (confirmation required), or web
  UI only, and give for each the exact command or path and a read-only
  verification command.
- **FR-028**: The recommended description MUST be one sentence naming the GitHub
  CLI account, Git identity, SSH alias, and `origin` remote, and "github.com
  only"; the recommended homepage MUST be
  `https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs` (verified reachable on
  2026-09-29) with "leave empty" as the alternative; recommended topics MUST be
  lowercase, hyphenated, at most twenty.
- **FR-029**: The social preview recommendation MUST include a source file
  `docs/assets/social-preview.svg` (text only, no external fonts or images,
  1280×640 viewBox) and the web UI upload path; it MUST state that no API
  exists for this setting.
- **FR-030**: The runbook MUST include the maintainer actions: enable private
  vulnerability reporting, set description/homepage/topics, optionally disable
  the empty wiki and projects, optionally enable Dependabot security updates,
  repair the v0.5.0 release notes from the changelog, and optionally tag
  `v0.5.1`; none of these are performed by the pull request.

**Drift tests**

- **FR-031**: A test MUST fail when any command's README reference block differs
  from `ghs <command> --help` output.
- **FR-032**: A test MUST fail when the README `go install` command does not use
  the module path from `go.mod` or when the stated Go version differs from the
  `go` directive (existing test retained).
- **FR-033**: A test MUST fail when the README release-archive section does not
  list every `goos`/`goarch` pair, the archive name template, or the checksums
  file name from `.goreleaser.yaml`.
- **FR-034**: A test MUST fail when a `make <target>` mentioned in `README.md` or
  `CONTRIBUTING.md` is not a `.PHONY` target of the `Makefile`.
- **FR-035**: A test MUST fail when a `## [x.y.z]` changelog section has no
  matching `[x.y.z]: https://github.com/izzamoe/ghs/releases/tag/vx.y.z`
  reference, or when the `[Unreleased]` reference does not compare from the
  newest section's tag.
- **FR-036**: A test MUST fail when a relative link in `README.md`,
  `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, or
  `CHANGELOG.md` targets a missing file, or when an absolute link's host is not
  in the allowlist (`github.com`, `pkg.go.dev`, `go.dev`, `cli.github.com`,
  `docs.github.com`, `keepachangelog.com`, `semver.org`,
  `www.contributor-covenant.org`). The test MUST NOT use the network.
- **FR-037**: A test MUST fail when any community file from FR-019 to FR-025 is
  missing or lacks its required phrases, and when `FUNDING.yml` exists.
- **FR-038**: A test MUST fail when the README troubleshooting section does not
  mention every `doctor` check name, or when the README lacks the config path or
  `Docs:` URL printed by `ghs --help`.
- **FR-039**: A test MUST fail when the CI workflow's gate commands differ from
  those documented in `CONTRIBUTING.md` (gofmt, vet, `go test -race`).
- **FR-040**: All drift tests MUST run inside `go test ./...`, need no network,
  and read repository files relative to the package directory as the existing
  `docs_test.go` does.

**Changelog and release**

- **FR-041**: `CHANGELOG.md` MUST gain link references for `0.5.0` and `0.5.1`
  and an `[Unreleased]` comparison from the newest section's tag, and an
  Unreleased entry describing this feature's user-visible changes (`ghs help
  <command>`, help `Docs:` line, README restructure, community files).
- **FR-042**: The release runbook MUST include verifying that the published
  release notes are non-empty and equal the changelog section, using
  `gh release view <tag> --json body`.

### Key Entities

- **Documentation set**: `README.md`, `CONTRIBUTING.md`, `SECURITY.md`,
  `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`, issue and PR templates;
  each with a source of truth it must agree with.
- **Source of truth**: the command table (help text), `go.mod` (module path, Go
  version), `.goreleaser.yaml` (assets), `Makefile` (targets),
  `.github/workflows/ci.yml` (gates), the `doctor` check names, the changelog
  headings.
- **Drift test**: a Go test that reads one document and one source of truth and
  fails on disagreement.
- **Metadata item**: a repository setting with a change channel (PR, admin
  API/CLI, web UI), a recommended value, and a verification command.
- **Maintainer action**: a change no pull request can make, recorded with its
  exact command and left to the maintainer.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A tester who has never used `ghs` completes install (either path),
  first run, and a `doctor` with zero `fail` lines using only the README, in
  under 20 minutes on each of Linux, macOS, and Windows.
- **SC-002**: 100% of registered commands and 100% of their flags appear in the
  README reference with text identical to the binary's help; `ghs help
  <command>` equals `ghs <command> --help` for all 15 commands.
- **SC-003**: Every one of the 14 troubleshooting rows and every one of the 10
  undo rows names a manual command that works without `ghs`; a reviewer
  verifies each against the fake-tool contract.
- **SC-004**: `gh api repos/izzamoe/ghs/community/profile` reports README,
  license, code of conduct, contributing, security policy, issue template, and
  pull request template all present after merge; `health_percentage` reaches
  100 once the maintainer sets the description.
- **SC-005**: Nine drift classes (FR-031 to FR-039) each have at least one test;
  introducing a one-character change in each source of truth makes exactly the
  corresponding test fail.
- **SC-006**: Zero invented facts: every absolute URL in the documentation set is
  on the allowlist and resolved on 2026-09-29; no package manager, website,
  email, funding link, response-time or compatibility promise appears.
- **SC-007**: The full suite still runs without network access and in under
  five minutes per platform in CI.
- **SC-008**: Every maintainer-only metadata action has an exact command and a
  verification command; after the maintainer runs them, `gh repo view` shows a
  non-empty description, at least eight topics, and the chosen homepage value.

## Assumptions

- The project has one maintainer (`@izzamoe`), no organization, no email
  address, no website, no funding account, and no package-manager distribution;
  none of these will be invented. If any appears later, the documentation and
  the link allowlist are updated in a reviewed change.
- pkg.go.dev and the Go module proxy serve `github.com/izzamoe/ghs` today
  (verified 2026-09-29: HTTP 200 for the module and `cmd/ghs` pages; proxy
  `@latest` is `v0.5.0`), so links to them are honest.
- The only supported host remains `github.com`; documentation does not describe
  GitHub Enterprise Server.
- Community files are in English; Contributor Covenant 2.1 is used verbatim
  with only the contact method filled in.
- The Spec Kit scripts accept `SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability`
  because the feature branch name does not follow the `NNN-name` pattern; the
  branch is not renamed.
- Admin-only settings are applied by the maintainer after reviewing the runbook;
  the pull request itself does not perform them and does not depend on them.
- `make` may be absent on Windows; every gate is documented in plain commands.
- Release archives already bundle `README.md`, `LICENSE`, and `CHANGELOG.md`
  (per `.goreleaser.yaml`); the README therefore avoids depending on files that
  are not in the archive for its core instructions.

## Out of Scope

- Any package-manager formula, tap, bucket, or distribution package.
- A documentation website, generated site, or wiki content.
- Localization.
- Changes to command behavior other than `ghs help <command>` and the `Docs:`
  help line.
- Branch protection rules or required status checks (recommended in the runbook
  as optional maintainer actions, not required by this feature).
- GitHub Discussions setup (optional maintainer action; `SUPPORT.md` routes
  questions to issues so that it works either way).
