# Specification Quality Checklist: ghs Account-Context Safety

**Purpose**: Validate specification completeness and quality before proceeding to planning
**Created**: 2026-09-28
**Updated**: 2026-09-28 (re-reviewed after scoped features A–E were made mandatory)
**Feature**: [spec.md](../spec.md)

**Review Ownership**: This checklist is a reviewer-owned requirements-quality review artifact. Mark an item `[x]` only when the reviewer determines the requirements-quality criterion is satisfied.
**Marker Semantics**: `[x]` means the criterion has been reviewed and satisfied for requirements quality. It does not mean implementation work is complete.

## Content Quality

- [x] CHK001 No implementation details (languages, frameworks, APIs)
  - Evaluated: the spec names `gh`, `git`, `ssh`, `ssh-keygen`, exit codes, and
    the Go version only where they are the user-facing contract or the subject of
    the requirement (Stories 10, 11, 14). Git's `includeIf`/`git config --global`
    (Story 8, FR-054/055) and the exact `ssh -T` invocation (FR-074) are named
    because the user-visible guarantee is precisely "only Git's own mechanism is
    used" and "the check is non-mutating"; they are contract, not design. No
    package names, function names, or library choices appear.
- [x] CHK002 Focused on user value and business needs
  - Evaluated: every story opens with a developer situation and a "Why this
    priority" tied to data loss, wrong-account risk, or user confusion. The five
    added stories (8, 12, 13, 14, 15) each state the user-facing failure they
    prevent (wrong-identity commits, silent key mismatch, unrecoverable deletion,
    undiagnosable rejected push, invisible active profile).
- [x] CHK003 Written for non-technical stakeholders
  - Evaluated: the stakeholders for a CLI are its developer users; scenarios are
    plain Given/When/Then with no code. A maintainer or product owner can read it
    without opening the source.
- [x] CHK004 All mandatory sections completed
  - Evaluated: User Scenarios & Testing (15 stories, 29 edge cases), Requirements
    (FR-001 to FR-079, Key Entities), Success Criteria (SC-001 to SC-012),
    Assumptions, Out of Scope are all filled; no template placeholder text remains.

## Requirement Completeness

- [x] CHK005 No [NEEDS CLARIFICATION] markers remain
  - Evaluated: `grep -c "NEEDS CLARIFICATION" spec.md` is 0. Open questions from
    the audit (workspace fate, GHES policy, license, breaking-change policy,
    whether `remove`/`doctor` are in scope, how `use` treats an unaliased origin)
    are all resolved in the spec body and Assumptions.
- [x] CHK006 Requirements are testable and unambiguous
  - Evaluated: each FR states a MUST with an observable outcome (file unchanged,
    exit code, call order, file content, exact output prefix). Validation rules
    give exact character sets and length limits (FR-007 to FR-012, FR-056).
    `doctor` outcomes are enumerated per check (FR-070 to FR-075); `list`/`status`
    columns and line formats are named (FR-077/078); `use --fix-remote` mutation
    and rollback order is fixed (FR-051).
- [x] CHK007 Success criteria are measurable
  - Evaluated: SC-001 (100 interrupted runs), SC-003 (30+ cases, 100% rejected),
    SC-004 (three platforms, under five minutes), SC-007 (six binaries), SC-009
    (eight mismatch classes), SC-011 (exactly one add / zero on rerun / one unset),
    SC-012 (exactly one warning, zero set-url) are counts or thresholds; SC-002,
    SC-005, SC-006, SC-008, SC-010 are pass/fail checks.
- [x] CHK008 Success criteria are technology-agnostic (no implementation details)
  - Evaluated: criteria speak of files, accounts, platforms, recorded tool calls,
    and tests, not of how they are implemented. SC-006 references the module's Go
    directive because the version mismatch is itself the defect being fixed.
- [x] CHK009 All acceptance scenarios are defined
  - Evaluated: every story has 3 to 11 Given/When/Then scenarios covering success,
    failure, and no-op paths; restoration-on-failure is covered for `use`
    (Stories 5 and 12), `init-ssh --upload`, and `import-all`; non-mutation is an
    explicit scenario for `doctor`, `status`, `list`, and `remove`.
- [x] CHK010 Edge cases are identified
  - Evaluated: 29 edge cases listed, covering malformed config, reserved names,
    alias collisions, missing public key, restoration failure, missing `origin`,
    missing home directory, missing GitHub CLI, Windows workspace paths, partially
    linked workspaces, `remove` while `origin` uses the alias, `doctor` offline and
    timeout, and `list` with accounts only on other hosts.
- [x] CHK011 Scope is clearly bounded
  - Evaluated: Out of Scope excludes GHES, SSH agent and `known_hosts` handling,
    non-origin remotes, migration of old SSH blocks, foreign `includeIf` entries,
    key deletion / GitHub CLI logout, and auto-repair from `doctor`. Story 7 makes
    the GHES boundary a tested rejection rather than an omission.
- [x] CHK012 Dependencies and assumptions identified
  - Evaluated: Assumptions cover GitHub CLI authentication, `github.com`-only,
    MIT license choice, breaking-change policy under major version 0, `go install`
    update path, shell-independent fakes for Windows, minimum Git version for
    `--fixed-value`, `gh auth status --json` support, and OpenSSH greeting
    semantics used by the `ssh auth` check.

## Feature Readiness

- [x] CHK013 All functional requirements have clear acceptance criteria
  - Evaluated: FR groups map to stories: FR-001–006 → Story 1; FR-007–014 →
    Story 2; FR-015–019 → Story 3; FR-020–025 → Story 4; FR-026–028 → Story 5;
    FR-029–032 → Story 6; FR-033–034 → Story 7; FR-035, FR-053–061 → Story 8;
    FR-036–039 → Story 9; FR-040–044 → Story 10; FR-045–048 → Story 11;
    FR-049–052 → Story 12; FR-062–066 → Story 13; FR-067–076 → Story 14;
    FR-077–079 → Story 15.
- [x] CHK014 User scenarios cover primary flows
  - Evaluated: add/import a profile, switch (with and without remote fix), init
    SSH with upload, clone, fix remote, link and unlink a workspace, remove a
    profile, diagnose with `doctor`, list/status, update, and contributor test run
    are all present.
- [x] CHK015 Feature meets measurable outcomes defined in Success Criteria
  - Evaluated: each SC is satisfiable by the FRs above it (SC-001 by FR-002,
    SC-002 by FR-021–027 and FR-051, SC-003 by FR-007–014, FR-036–038, and
    FR-056, SC-004 by FR-042/045–047, SC-005 by FR-044, SC-006 by FR-040, SC-007
    by FR-043, SC-008 by the planning phase producing `tasks.md`, SC-009 by
    FR-067–076, SC-010 by FR-063, SC-011 by FR-054/057/058, SC-012 by FR-049).
- [x] CHK016 No implementation details leak into specification
  - Evaluated: same finding as CHK001; the only tool names are those the user
    types, that appear in user-visible output, or whose exact use is the
    guarantee being specified.

## Scoped Features A–E Are Mandatory

- [x] CHK022 (A) `ghs remove <profile>` with safety behavior is a mandatory story
  - Evaluated: Story 13 (P2) with 8 scenarios; FR-062–066; SC-010. Safety model
    is explicit: nothing unrecoverable is deleted (no keys, no SSH config edits,
    no GitHub CLI logout/switch, no Git identity writes), output lists what
    remains, workspace unlink happens first and aborts the removal on failure.
- [x] CHK023 (B) `ghs doctor` checks every required dimension non-mutatingly
  - Evaluated: Story 14 (P2) with 11 scenarios; FR-067–076 cover GitHub CLI
    active identity (FR-070), current Git identity with origin and scope (FR-071),
    origin/alias compatibility (FR-072), SSH alias config (FR-073), and safe
    non-mutating SSH authentication verification with the exact command and
    forbidden options (FR-074); non-mutation is FR-068 and Scenario 10; SC-009.
- [x] CHK024 (C) `list`/`status` identify the active profile and each mismatch
  - Evaluated: Story 15 (P2) with 10 scenarios; FR-077 (columns, `*` marker,
    trailing active line, degraded mode) and FR-078 (three resolutions and six
    enumerated `mismatch:` conditions); FR-079 degraded operation without the
    GitHub CLI.
- [x] CHK025 (D) `use` warns on an unaliased `origin` and offers `--fix-remote`
  - Evaluated: Story 12 (P1) with 8 scenarios; FR-049 (single stderr warning,
    both remedies named, exit 0), FR-050 (preflight rejection before any
    mutation), FR-051 (mutation order and reverse-order rollback), FR-052
    (already-correct path); SC-012. Both the warning and the explicit flow are
    required, not alternatives.
- [x] CHK026 (E) `workspace` is implemented as safe Git `includeIf gitdir`
      automation, not removed
  - Evaluated: Story 8 rewritten (P2) with 11 scenarios; FR-035 retains the flag
    and key; FR-053–061 define the identity file, the `git config --global --add`
    / `--unset --fixed-value` mechanism, the pattern rules, path validation,
    idempotence, preservation on overwrite, and non-nesting; FR-055 forbids
    touching `~/.gitconfig` directly; SC-011. Story 10 Scenario 3, FR-005, the
    Assumptions, and Out of Scope no longer describe removal.

## Existing Bug Fixes Preserved

- [x] CHK027 Every confirmed defect fix from the audit is still specified
  - Evaluated by diffing against the previous revision: Stories 1–7 and 9–11 are
    unchanged except for additive edits (Story 9 Scenario 5 lists the new
    commands; Story 10 Scenario 3 lists the new changes; Story 11 adds the fake
    `ssh`). FR-001–004, FR-006–034, FR-036–048 are textually unchanged except
    FR-029 (adds `use --fix-remote`), FR-045/046 (add `ssh` and extra fake
    state), and FR-005 (drops the `workspace` exception). No fix was weakened.

## Constitution Alignment (v1.1.0)

- [x] CHK017 Principle I (no state loss): FR-001–003, FR-017, FR-019, FR-021–027,
      FR-051, FR-054, FR-058, FR-062–064, FR-068
  - Evaluated: preflight-before-mutate, atomic write, append-only SSH config,
    account restoration, reverse-order rollback for `use --fix-remote`, atomic
    identity file, non-destructive `remove`, non-mutating `doctor` each have a
    requirement and a scenario.
- [x] CHK018 Principle II (test-first, hermetic): FR-045–048, Story 11
  - Evaluated: fake tools (now including `ssh`), isolated home and config,
    network-free, call-order assertions are required; SC-008 requires FR-to-test
    traceability; SC-009–012 are defined in terms of the recorded invocation log.
- [x] CHK019 Principle III (explicit CLI contract): FR-036–039, FR-044, FR-067,
      FR-077, FR-078
  - Evaluated: strict flags, exit codes `0`/`1`/`2`, help/README parity, and
    fixed output formats for `doctor`, `list`, and `status` are required with
    scenarios.
- [x] CHK020 Principle IV (simplicity, github.com only): FR-033–034, FR-035,
      FR-053–055
  - Evaluated: GHES rejected as usage error. The `workspace` key is no longer a
    stored-but-unused value: FR-053 requires every reader to act on it, so the
    "dead options are removed" rule is satisfied by implementation rather than
    removal, using only Git's built-in mechanism and no new dependency.
- [x] CHK021 Principle V (release discipline): FR-040–043
  - Evaluated: changelog (now listing the new commands and flags), license,
    version parity, CI on three platforms, tagged releases with binaries.

## Notes

- All 27 items reviewed on 2026-09-28 against spec.md as amended; none require
  clarification before `/speckit-plan`.
- Reviewer decisions recorded in the spec rather than left open: `workspace` is
  implemented via Git conditional includes (not removed); GHES is rejected (not
  supported); license is MIT; stricter flags are accepted as a breaking change
  under 0.x; `remove` is reversible by design; `doctor` reports and never repairs.
- Items marked incomplete require spec updates before `/speckit-clarify` or `/speckit-plan`
