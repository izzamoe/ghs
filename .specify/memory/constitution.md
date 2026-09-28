<!--
Sync Impact Report
- Version change: 1.0.0 → 1.1.0
- Modified principles:
  - I. User State Is Never Lost — added: temporary account switches must be restored;
    files not owned by ghs are read-modify-append only; every mutation names its target.
  - III. Explicit CLI Contract — added: flags are position-independent but a flag that
    takes a value MUST receive one; the same flag MUST NOT be accepted twice.
  - IV. Simplicity and Standard Library First — added: a host other than github.com is a
    usage error until per-host support exists end-to-end; stored-but-unused config keys
    are removed rather than kept.
- Added sections: none
- Removed sections: none
- Templates checked:
  - .specify/templates/plan-template.md ✅ Constitution Check gate section is generic and compatible
  - .specify/templates/spec-template.md ✅ no principle-specific placeholders required
  - .specify/templates/tasks-template.md ✅ test-first ordering matches Principle II
  - .specify/templates/checklist-template.md ✅ no change required
- Follow-up TODOs: none
-->

# ghs Constitution

## Core Principles

### I. User State Is Never Lost

`ghs` mutates state that the user relies on every day: the `ghs` config file, the active
GitHub CLI account, Git identity (local or global), `~/.ssh/config`, and repository
remotes. Therefore:

- Every command MUST check all preconditions it can check before performing its first
  mutation (preflight before mutate).
- A command that performs more than one mutation MUST either complete all of them or
  restore the earlier ones and report both the original failure and the restore result.
- Existing files MUST never be truncated or replaced with partial content. Writes to
  files owned by `ghs` MUST be atomic (write to a sibling temporary file, then rename).
- A config file that cannot be read or parsed MUST cause the command to stop with an
  actionable error; it MUST NOT be treated as empty and overwritten.
- Files not owned by `ghs` (for example `~/.ssh/config`) MUST only be appended to, never
  rewritten, and MUST be left untouched when the required content is already present.
- A command that switches the active GitHub CLI account only as a means to another end
  (importing, uploading a key) MUST restore the previously active account before it
  exits, on success and on failure, and MUST report if restoration fails.
- Every mutation MUST be attributable: the command output names each file, account,
  or remote it changed, so the user can undo it by hand if needed.

Rationale: a small helper that silently destroys a user's config or leaves them halfway
between two GitHub identities is worse than no helper at all.

### II. Test-First and Hermetic (NON-NEGOTIABLE)

- Every behavior change starts with a failing test (unit or end-to-end), which is then
  made to pass. Red → Green → Refactor is mandatory for defect fixes.
- The automated test suite MUST run without network access, without a real GitHub
  account, without real SSH keys, and without touching the developer's real `$HOME`,
  `~/.ssh`, `~/.gitconfig`, or GitHub CLI state.
- Interactions with external tools (`gh`, `git`, `ssh-keygen`) MUST be covered by
  end-to-end tests that execute the real `ghs` binary against fake tool executables placed
  on `PATH`, which record every invocation and answer from scripted state.
- Unit tests remain the primary tool for parsing, validation, and URL logic; end-to-end
  tests prove command ordering, rollback, and exit codes.

Rationale: the defects this project fixes are ordering and state-handling bugs that unit
tests alone cannot catch, and contributors must be able to run the full suite anywhere.

### III. Explicit CLI Contract

- Every command documents its positional arguments and flags in `ghs --help` and in
  `README.md`; the two MUST agree with the implementation.
- Unknown flags, unknown commands, missing required values, and surplus positional
  arguments MUST be rejected with a usage error and a non-zero exit code. Nothing is
  silently ignored.
- Success output goes to stdout; errors go to stderr prefixed with `ghs:`; exit code `0`
  means success, `2` means usage error, `1` means any other failure.
- Flags MAY appear in any position after the command name, but a flag that takes a
  value MUST receive a non-empty value, the same flag MUST NOT be given twice, and a
  boolean flag MUST NOT be given a value. Violations are usage errors.
- A documented flag, config key, or command MUST have implemented, tested behavior. Dead
  options are removed rather than left in place.
- Input that is written into files interpreted by other tools (config file, SSH config,
  Git remote URLs) MUST be validated against an explicit allowed character set or
  correctly quoted before it is written.

Rationale: users trust a switch tool with credentials and identity; a typo that is
silently accepted becomes a wrong commit author or a key uploaded to the wrong account.

### IV. Simplicity and Standard Library First

- The module MUST have no third-party runtime dependencies unless a design decision
  recorded in a feature's `research.md` justifies one.
- Keep the package layout flat: one `cmd/ghs` entry point and small `internal/` packages
  with a single responsibility (`cli`, `config`, `ghops`, `gitops`, `sshops`, `runner`).
- Prefer removing a feature over keeping an unimplemented or half-implemented one.
- Only `github.com` is a supported host unless a feature explicitly adds and tests
  per-profile host support end-to-end. Until then, any other host value is a usage
  error, never a silent fallback to `github.com` behavior.
- A config key or flag whose value no command reads MUST be removed from the CLI and
  documentation rather than stored indefinitely.

Rationale: the tool is a few hundred lines; every added abstraction or dependency is a
maintenance cost for a single maintainer and a security surface for every user.

### V. Release Discipline

- Versions follow Semantic Versioning with `vMAJOR.MINOR.PATCH` git tags on `main`.
  While the major version is `0`, a MINOR bump MAY contain breaking CLI changes, and the
  changelog MUST call them out.
- `CHANGELOG.md` (Keep a Changelog format) MUST be updated in the same change that alters
  user-visible behavior. Existing tags without release notes are recorded as historical.
- Every tag push MUST produce a GitHub Release with notes and prebuilt binaries via CI;
  `go install github.com/izzamoe/ghs/cmd/ghs@latest` remains the primary install path.
- The Go version stated in documentation MUST match the `go` directive in `go.mod`, and
  CI MUST use the version from `go.mod`.
- `main` MUST be green: formatting, `go vet`, and the full test suite with the race
  detector on Linux, macOS, and Windows.

Rationale: users install this tool with `go install @latest` and `ghs update`; a broken
or undocumented release reaches them immediately.

## Additional Constraints

- **Language/toolchain**: Go, version pinned by `go.mod`; no cgo.
- **Supported platforms**: Linux, macOS, Windows (amd64 and arm64). Path handling MUST use
  `path/filepath`; SSH config paths MUST be written with forward slashes and quoted when
  they contain whitespace.
- **Security**: `ghs` MUST never read, print, or upload private key material; only the
  `.pub` file is passed to `gh ssh-key add`. Generated keys and the config file are
  created with mode `0600`; directories with `0700`. `ghs` MUST refuse nothing on the basis
  of user id but MUST document that it must not be run with `sudo`.
- **Configuration**: single file at `$XDG_CONFIG_HOME/ghs/config.conf` or
  `~/.config/ghs/config.conf`, INI-like, human-editable; unknown keys are ignored on load
  so newer files do not break older binaries.
- **External tools**: `gh`, `git`, and `ssh-keygen` are invoked as subprocesses with
  argument vectors only (never through a shell).

## Development Workflow

- Non-trivial changes go through the Spec Kit sequence in `specs/<NNN-name>/`:
  `spec.md` → `plan.md` (+ `research.md`, `data-model.md`, `contracts/`, `quickstart.md`)
  → `tasks.md` → implementation. Bug fixes touching more than one command count as
  non-trivial.
- Pull request gates (enforced by CI): `gofmt -l` reports nothing, `go vet ./...` passes,
  `go test -race ./...` passes on all three platforms, README Go version matches `go.mod`.
- Every task in `tasks.md` names the exact files it touches and, for behavior changes,
  the test that must fail first.
- Reviewers verify compliance with Principles I–III explicitly for any change that
  mutates user state or parses user input.

## Governance

- This constitution supersedes ad-hoc practice for the `ghs` repository. Conflicts
  between this document and other guidance are resolved in favor of this document.
- Amendments are made by editing this file in a pull request that states the version
  bump and rationale in the Sync Impact Report comment at the top of the file.
- Versioning of this document: MAJOR for removing or redefining a principle, MINOR for
  adding a principle or materially expanding guidance, PATCH for clarifications.
- Compliance review: `/speckit-plan` records a Constitution Check for every feature;
  `/speckit-analyze` treats any violation of a MUST statement as CRITICAL.

**Version**: 1.1.0 | **Ratified**: 2026-09-28 | **Last Amended**: 2026-09-28
