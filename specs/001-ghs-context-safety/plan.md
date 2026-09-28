# Implementation Plan: ghs Account-Context Safety

**Branch**: `001-ghs-context-safety` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/001-ghs-context-safety/spec.md`

## Summary

Make every `ghs` command safe with respect to the five pieces of state it touches
(its own config, the active GitHub CLI account, Git identity, `~/.ssh/config`,
and the `origin` remote), and add the commands a user needs to see and repair
that state: `remove`, `doctor`, `workspace` (Git conditional-include automation),
`use --fix-remote`, and active-profile/mismatch reporting in `list` and `status`.
Technical approach: keep the flat standard-library-only layout; add a strict
argument parser and a usage-error exit code; make config persistence atomic and
strict; centralize validation, account switching with restore, and remote URL
classification; drive every behavior from hermetic end-to-end tests that run the
real binary against fake `gh`, `git`, `ssh`, and `ssh-keygen` executables; and
add local quality gates, CI on three platforms, and a tag-driven release. All
design decisions are recorded in [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.26.3 (the `go` directive in `go.mod`); no cgo

**Primary Dependencies**: standard library only at runtime. External tools
invoked as subprocesses with argument vectors: `gh` (≥ a version with
`auth status --json hosts`, already required), `git` (≥ 2.30 for
`config --fixed-value`), `ssh` (OpenSSH-compatible), `ssh-keygen`. Build-time
only: GoReleaser in the release workflow.

**Storage**: INI-like config file at `$XDG_CONFIG_HOME/ghs/config.conf` or
`~/.config/ghs/config.conf` (atomic replace); `ghs`-owned identity files
`gitconfig-<profile>` in the same directory; append-only writes to
`~/.ssh/config`; global Git config through `git config --global` only.

**Testing**: `go test` (standard `testing` package, no assertion library);
unit tests per package; end-to-end tests in `internal/e2e` executing the built
binary with the test binary doubling as every fake tool; race detector on in CI.

**Target Platform**: Linux, macOS, Windows; amd64 and arm64.

**Project Type**: single CLI binary (`cmd/ghs`) with `internal/` packages.

**Performance Goals**: every command completes in under two seconds excluding
network round-trips; `doctor`'s only network call is bounded by
`ConnectTimeout=10`; the full test suite runs in under five minutes per
platform (SC-004).

**Constraints**: no third-party runtime dependencies (Principle IV); never
invoke a shell; never read, print, or upload private key material; tests must
not touch the developer's `$HOME`, `~/.ssh`, `~/.gitconfig`, or GitHub CLI
state and must pass with the network disabled (Principle II).

**Scale/Scope**: 15 commands, roughly 2.5k lines of Go after the change,
roughly 120 end-to-end and unit tests, one maintainer.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Pre-research | Post-design |
|-----------|------|--------------|-------------|
| I. User State Is Never Lost | Every command preflights before its first mutation; multi-mutation commands complete or roll back and report both results; `ghs`-owned files written atomically; unreadable config stops the command; non-owned files append-only; temporary account switches restored; every mutation named in output | PASS: FR-001–003, FR-017, FR-019–028, FR-051, FR-054, FR-058, FR-062–064, FR-068 | PASS: research R1 (atomic rename), R4 (`git config --global` only; identity file owned by `ghs`), R5 (`doctor` cannot write `known_hosts`), R12 (`remove` reversible). `add-profile --workspace` link failure after save is reported with the rerun command rather than rolled back because the saved value is correct and the link step is idempotent (same reasoning the spec already applies to `clone`'s SSH block) |
| II. Test-First and Hermetic | Failing test first for every behavior change; suite needs no network, no real accounts, no real `$HOME`; external tools covered by end-to-end tests against fakes on `PATH` | PASS: FR-045–048, Story 11 | PASS: research R10; every user-story phase in `tasks.md` lists its tests before its implementation tasks, and each implementation task names the test that must fail first |
| III. Explicit CLI Contract | Help and README agree with the implementation; unknown flags/commands/surplus args rejected; exit `0`/`1`/`2`; `ghs:` prefix on stderr; position-independent flags with value/duplicate rules; no dead options; validated or quoted input to other tools | PASS: FR-036–039, FR-044, FR-007–015 | PASS: research R2/R3; [contracts/cli.md](./contracts/cli.md) is the single source for README, help text, and test assertions; `docs_test.go` enforces parity |
| IV. Simplicity and Standard Library First | No third-party runtime dependency; flat layout with `cmd/ghs` and single-purpose `internal/` packages; remove rather than keep half-implemented features; `github.com` only; no stored-but-unused keys | PASS with note: `workspace` is retained *because it is implemented* (FR-053 requires every reader to act on it); the constitution's rule targets keys "whose value no command reads", which no longer applies | PASS: no new dependency; new files stay inside existing packages plus test-only `internal/e2e`; one build-time helper `internal/tools/changelog` (see Complexity Tracking) |
| V. Release Discipline | SemVer tags; changelog in the same change; tag push produces a release with binaries; README Go version matches `go.mod`; `main` green on three platforms | PASS: FR-040–043 | PASS: research R11; CI matrix uses `go-version-file: go.mod`; release notes extracted from `CHANGELOG.md` |
| Additional constraints | Go pinned by `go.mod`; `path/filepath` for paths; SSH paths forward-slashed and quoted when containing whitespace; keys and config `0600`, directories `0700`; no `sudo` documented; single config file; unknown keys ignored on load; tools invoked without a shell | PASS | PASS: FR-003, FR-005, FR-015, FR-054, R14 |

No gate fails. Nothing requires an unresolved clarification marker; the two
judgment calls (workspace retention, `add-profile --workspace` link failure
handling) are recorded above and in research R4.

## Project Structure

### Documentation (this feature)

```text
specs/001-ghs-context-safety/
├── spec.md              # Feature specification (amended 2026-09-28)
├── plan.md              # This file
├── research.md          # Phase 0: decisions R1–R14
├── data-model.md        # Phase 1: entities, validation, workspace link states
├── quickstart.md        # Phase 1: validation guide
├── contracts/
│   ├── cli.md           # Phase 1: command grammar, exit codes, exact output
│   └── fake-tools.md    # Phase 1: fake tool behavior and state schema
├── checklists/
│   └── requirements.md  # Spec quality review (27 items)
└── tasks.md             # Phase 2: dependency-ordered tasks with test names
```

### Source Code (repository root)

```text
cmd/ghs/
└── main.go                      # maps cli.UsageError → exit 2, other errors → exit 1

internal/cli/
├── app.go                       # command table (cmdSpec per command), dispatch, help text, --help
├── args.go                      # strict parser: parseArgs(spec, args) → positionals, flags, *UsageError   [new]
├── usage.go                     # UsageError type, per-command usage text                                   [new]
├── resolve.go                   # resolve login/email/alias → profile; shared by list/status/doctor/use     [new]
├── add_profile.go               # add-profile, add-from-gh, import-all (strict load, validation, workspace hook)
├── set_email.go                 # validation, identity-file refresh
├── list.go                      # ACTIVE marker, WORKSPACE and GH AUTH columns, trailing account line
├── status.go                    # three resolutions + mismatch lines
├── doctor.go                    # six checks, summary, exit code                                            [new]
├── remove.go                    # safe removal                                                              [new]
├── workspace.go                 # link / --unlink                                                           [new]
├── use.go                       # preflight, ordered mutations, rollback, origin warning, --fix-remote
├── clone.go                     # host check before mutation, upload while switched
├── init_ssh.go                  # read SSH config first, upload with restore
├── fix_remote.go                # classifier-based rewrite, already-correct path, missing-block warning
├── update.go                    # strict args only
├── args_test.go                 # parser table tests                                                       [new]
├── docs_test.go                 # README ↔ help parity, README ↔ go.mod version, github.com-only text     [new]
├── resolve_test.go              # resolver unit tests                                                       [new]
└── add_profile_test.go          # existing tests kept and extended

internal/config/
├── profile.go                   # Profile{..., Workspace, Extra []KeyValue}, Config helpers
├── load.go                      # strict parser with line numbers; missing file → empty config
├── save.go                      # atomic replace, mode preservation, Extra keys
├── validate.go                  # ValidateName/Login/GitName/Email/KeyPath/Alias/Workspace, CheckUnique     [new]
├── identity.go                  # IdentityFilePath, WriteIdentityFile (atomic), ReadIdentityFile           [new]
├── paths.go                     # DefaultPath, ExpandPath (unchanged)
├── load_test.go, save_test.go, validate_test.go, identity_test.go, windows_test.go

internal/ghops/
├── user.go                      # + ActiveAccount, IsAuthenticated, WithAccount, AddSSHKey
└── user_test.go

internal/gitops/
├── git.go                       # InRepo, SetIdentity (step-aware), IdentityWithOrigin, OriginURL, SetOriginURL, Clone
├── remote.go                    # ParseRemote, Remote{Kind,Host,Path}, RewriteGitHubURL, CloneURL, CloneDirectory   [new]
├── includeif.go                 # WorkspacePattern, IncludeIfKey, HasInclude, AddInclude, RemoveInclude             [new]
├── git_test.go, remote_test.go, includeif_test.go

internal/sshops/
├── ssh.go                       # EnsureKey (pub check), EnsureConfig (quoting, newline, read-first), UploadKey
├── parse.go                     # ParseHostBlock for doctor                                                 [new]
├── auth.go                      # Verify(alias) → classification of ssh -T output                           [new]
├── ssh_test.go, parse_test.go, auth_test.go, windows_test.go

internal/runner/
├── runner.go                    # + LookPath, CombinedOutput, RunIn
└── runner_test.go                                                                                            [new]

internal/e2e/                    # test-only package                                                          [new]
├── main_test.go                 # TestMain: fake dispatch by argv[0], build ghs once
├── harness_test.go              # sandbox: HOME, XDG_CONFIG_HOME, bin/, state, log, run, assertions
├── fake_state_test.go           # state schema, load/save, log append
├── fake_gh_test.go, fake_git_test.go, fake_ssh_test.go, fake_keygen_test.go
├── fakes_selftest_test.go       # fakes behave per contracts/fake-tools.md
├── isolation_test.go            # US11
├── config_test.go               # US1
├── validation_test.go           # US2
├── sshconfig_test.go            # US3
├── upload_test.go               # US4
├── use_test.go                  # US5
├── use_remote_test.go           # US12
├── remote_test.go               # US6
├── hosts_test.go                # US7
├── flags_test.go                # US9
├── workspace_test.go            # US8
├── remove_test.go               # US13
├── list_test.go, status_test.go # US15
└── doctor_test.go               # US14

internal/tools/changelog/
└── main.go                      # prints the CHANGELOG.md section for a tag (release notes)               [new]

Makefile                         # fmt, vet, test, e2e, check                                                [new]
.github/workflows/ci.yml         # matrix Linux/macOS/Windows, go-version-file: go.mod                       [new]
.github/workflows/release.yml    # on v* tags: GoReleaser with changelog notes                               [new]
.goreleaser.yaml                 # 6 targets                                                                 [new]
CHANGELOG.md, LICENSE            # Keep a Changelog; MIT                                                     [new]
README.md                        # rewritten command list, Go version, exit codes, github.com-only, no sudo
```

**Structure Decision**: single Go module, one binary, existing `internal/`
packages extended in place; the only new packages are the test-only
`internal/e2e` and the build-time `internal/tools/changelog`. This keeps the
constitution's flat layout and lets each package's unit tests stay next to the
code while the end-to-end suite observes the whole binary.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Second `main` package `internal/tools/changelog` (constitution prefers one `cmd/ghs` entry point) | Release notes must be extracted from `CHANGELOG.md` on every tag; a Go program runs identically on the release runner and on a Windows contributor's machine | A shell/`awk` script would not run for Windows contributors verifying the release locally; GoReleaser's built-in changelog uses commit messages, not the curated `CHANGELOG.md` the constitution requires |

## Phase 0: Research (complete)

See [research.md](./research.md): R1 atomic writes, R2 strict parser, R3 exit
codes, R4 workspace via `includeIf`, R5 non-mutating SSH auth check, R6 GitHub
CLI interfaces, R7 Git identity provenance, R8 remote classification, R9 SSH
config parsing and quoting, R10 hermetic harness, R11 quality gates and CI,
R12 `remove` safety model, R13 degraded mode without `gh`, R14 Windows.

## Phase 1: Design (complete)

- [data-model.md](./data-model.md): Profile, Config file, Identity file,
  Workspace link (with its six observable states), SSH config block, GitHub CLI
  account set, Repository remote, Doctor check, Mismatch, Tool invocation log,
  Fake state.
- [contracts/cli.md](./contracts/cli.md): full command list, exit codes,
  per-command preflight/mutation/rollback order, exact stdout/stderr formats.
- [contracts/fake-tools.md](./contracts/fake-tools.md): fake tool behavior,
  state schema, invocation log, sandbox helper API.
- [quickstart.md](./quickstart.md): how to build, run the gates, run the
  end-to-end suite, drive the binary manually against the fakes, and smoke-test
  read-only commands on a real machine.

Constitution re-check after design: all gates still pass (table above).

## Phase 2: Tasks

Generated in [tasks.md](./tasks.md): setup, foundational, one phase per user
story in priority order (P1 stories first), polish, and an FR-to-test
traceability matrix for SC-008.
