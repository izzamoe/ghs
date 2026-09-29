# Implementation Plan: ghs Public Discoverability and Contributor Readiness

**Branch**: `docs/public-discoverability` (Spec Kit feature `002-public-discoverability`; run scripts with `SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability`) | **Date**: 2026-09-29 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `/specs/002-public-discoverability/spec.md`

## Summary

Make `github.com/izzamoe/ghs` legible to a new user, a user in trouble, and a
first contributor without inventing anything: rewrite `README.md` around install
(two real paths, no package manager), first run, a verbatim per-command
reference, troubleshooting with manual undo, config/data privacy, and
cross-platform notes; add the GitHub community health files (CONTRIBUTING,
SECURITY, CODE_OF_CONDUCT, SUPPORT, issue forms, PR template, Dependabot
config, no FUNDING); record every admin-only repository metadata change as a
maintainer action with its exact command and verification; make `ghs help
<command>` work and add a `Docs:` line to help; and bind every checkable
documentation claim to a test that runs in `go test ./...` without network.
Design decisions and the read-only audit are in [research.md](./research.md).

## Technical Context

**Language/Version**: Go 1.26.3 (the `go` directive in `go.mod`); no cgo. Local
audit toolchain was go1.27.1, which satisfies the directive.

**Primary Dependencies**: standard library only (`regexp`, `net/url`, `os`,
`path/filepath`, `strings`, `testing`). External tools named in documentation:
`gh`, `git` ≥ 2.30, OpenSSH client, `go` (for `go install`/`ghs update`),
`sha256sum`/`shasum`/`Get-FileHash` for checksum verification. Build-time:
GoReleaser v2 (unchanged).

**Storage**: none at runtime. Documents are files in the repository; see
[data-model.md](./data-model.md) for the document/source-of-truth map.

**Testing**: `go test` (standard `testing`); documentation drift tests in
`internal/cli` (which already reads `../../README.md`) and
`internal/tools/changelog`; one new e2e test in `internal/e2e/flags_test.go`
for `ghs help <command>`; race detector in CI as today.

**Target Platform**: Linux, macOS, Windows; amd64 and arm64 (documentation must
be correct on all three; tests must pass on all three, including path
separators and CRLF checkouts, which `.gitattributes` prevents).

**Project Type**: single CLI binary plus repository documentation.

**Performance Goals**: drift tests add under one second to `go test
./internal/cli`; total suite stays under five minutes per platform (SC-007).

**Constraints**: no network in tests (link test checks hosts against an
allowlist, never fetches); no third-party dependency; no new runtime behavior
except `help <command>` and the `Docs:` line; no invented channel, site,
contact, funding, or promise (constitution VI); admin-only metadata is
documented, not executed, by the pull request; git state is not changed by the
planning work (no commits, no branch rename, no tags).

**Scale/Scope**: 15 commands; README grows from 298 to roughly 600 lines; 7 new
community files plus 1 SVG; roughly 12 new tests; 2 source edits of a few lines
each in `internal/cli/app.go`.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Pre-research | Post-design |
|-----------|------|--------------|-------------|
| I. User State Is Never Lost | No new mutation; documentation of every existing mutation and its manual undo | PASS: FR-012, FR-014 | PASS: the only source change (`help <command>` rewrite in `App.Run`, `Docs:` line) runs before any tool call and mutates nothing; research R4 lists the undo for each of the 10 mutations, all using `gh`/`git`/editor |
| II. Test-First and Hermetic | Behavior change starts with a failing test; suite needs no network | PASS: FR-008, FR-031–040 | PASS: tasks T004–T016 write the failing tests first; the link test parses URLs and never fetches (R9); the e2e `help <cmd>` test uses the existing sandbox |
| III. Explicit CLI Contract | Help and README agree; nothing silently ignored | PASS: FR-006–010 | PASS: README reproduces `usageText()` verbatim (R2); `help <unknown>` becomes the unknown-command usage error instead of silently printing general help (R3); [contracts/docs.md](./contracts/docs.md) §1 fixes the help footer |
| IV. Simplicity and Standard Library First | No dependency; no half-features | PASS | PASS: no Markdown parser, no generator, no website; string/regexp tests only (R9) |
| V. Release Discipline | Changelog in the same change; release notes non-empty | PASS: FR-041–042 | PASS: `[Unreleased]` entry and link references (R10); runbook step verifies `gh release view --json body`; the v0.5.0 notes repair is a maintainer action |
| VI. Honest, Drift-Free Public Documentation | No invented channel/site/contact/funding/guarantee; checkable claims tested; allowlisted hosts; community files via PR; admin metadata as recorded maintainer actions; recovery without `ghs` | PASS: FR-001, FR-020–021, FR-025, FR-027–030, FR-031–039 | PASS: R1 (two real install paths), R6 (no email), R7 (channel table with commands), R8 (verified homepage), R9 (allowlist in `links_test.go`), R4 (manual undo) |
| Additional constraints | Config path and permissions documented as implemented; `sudo` warning; tools invoked without shell | PASS | PASS: R5 lists files, modes, and network calls taken from source; R12 platform notes taken from `sshops`/`config` tests |

No gate fails. Two judgment calls are recorded: (1) the security route is
GitHub's private vulnerability reporting, which is currently disabled and must
be enabled by the maintainer before `SECURITY.md` merges; the file also gives a
fallback that works while it is disabled (R6). (2) The changelog test requires
`[Unreleased]` to compare from the newest *section* (`0.5.1`). No `v0.5.1` tag
exists, and implementation verified that the `[0.5.1]` release link and the
`compare/v0.5.1...HEAD` link return HTTP 404 until one does. Because `v0.5.0`
already points at `dd041a4` (the commit that added the `0.5.1` section), the
runbook's maintainer action M6 tags exactly `dd041a4` as `v0.5.1` before merge,
or applies the recorded fallback (fold the section into `0.5.0`, compare from
`v0.5.0`); tagging a later `main` commit would publish Unreleased changes under
notes that do not mention them (R10).

## Project Structure

### Documentation (this feature)

```text
specs/002-public-discoverability/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0: audit A1–A7 and decisions R1–R14
├── data-model.md        # Phase 1: documents, sources of truth, drift checks, metadata items
├── quickstart.md        # Phase 1: how to validate the feature locally and after merge
├── contracts/
│   └── docs.md          # Phase 1: README outline, exact strings, community file content, help contract, runbook table
├── checklists/
│   └── requirements.md  # Spec quality review (25 items)
└── tasks.md             # Phase 2: dependency-ordered tasks with file paths and verification
```

### Source Code (repository root)

```text
README.md                                  # rewritten per contracts/docs.md §1 (sections, order, exact strings)
CONTRIBUTING.md                            # new: contributor guide + maintainer runbook (release, repository settings)
SECURITY.md                                # new: supported versions, private reporting, fallback, scope, no bounty
SUPPORT.md                                 # new: self-help order, issue route, best-effort statement
CODE_OF_CONDUCT.md                         # new: Contributor Covenant 2.1 with GitHub-based contact
CHANGELOG.md                               # link references for 0.5.0/0.5.1, Unreleased base, Unreleased entry
docs/assets/social-preview.svg             # new: text-only 1280×640 source for the social preview
.github/ISSUE_TEMPLATE/bug_report.yml      # new: issue form
.github/ISSUE_TEMPLATE/feature_request.yml # new: issue form
.github/ISSUE_TEMPLATE/config.yml          # new: blank issues off, contact links
.github/PULL_REQUEST_TEMPLATE.md           # new: gate checklist
.github/dependabot.yml                     # new: github-actions weekly

internal/cli/
├── app.go                                 # Run: `help <command>` → `<command> --help`; printHelp: `Docs:` line   [2 small edits]
├── docs_test.go                           # + TestReadmeCommandReferenceMatchesHelp, TestReadmeInstallUsesModulePath,
│                                          #   TestReadmeReleaseArchivesMatchGoreleaser, TestReadmeMatchesHelpFooter,
│                                          #   TestReadmeTroubleshootingCoversDoctorChecks, TestReadmeUsesNeutralExamples,
│                                          #   TestHelpCommandAliasMatchesFlag, TestHelpUnknownTopicIsUsageError
├── community_test.go                      # new: TestCommunityFilesPresentAndComplete, TestDocsMakeTargetsExist,
│                                          #   TestContributingMatchesCIGates, TestDocsMakeNoInventedClaims,
│                                          #   TestSecuritySupportedVersionMatchesChangelog
└── links_test.go                          # new: TestDocLinksResolve (files + anchors), TestDocURLHostsAllowed,
                                           #   TestHeadingAnchor, allowedLinkHosts

internal/tools/changelog/
└── main_test.go                           # + TestChangelogLinkReferences

internal/e2e/
└── flags_test.go                          # + TestFlags_HelpCommandPerCommand, TestFlags_HelpUnknownTopicExit2
```

**Structure Decision**: no new packages. Documentation tests stay in
`internal/cli` because it already owns README parity tests, reads files
relative to `../..`, and can call `usageText()` on the unexported command
table. The changelog test sits beside the extractor it complements. Community
files live where GitHub looks for them (repository root and `.github/`).

## Complexity Tracking

> No constitution violations. Nothing to justify.

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| none | | |

## Phase 0: Research (complete)

See [research.md](./research.md): Part A audit (A1 metadata, A2 releases and
tags, A3 README gaps, A4 help, A5 existing tests, A6 build/CI/release, A7 Spec
Kit) and Part B decisions (R1 install paths, R2 verbatim reference, R3 `help
<command>` and `Docs:`, R4 troubleshooting and undo tables, R5 config and data,
R6 community files without invented contacts, R7 metadata channel table, R8
recommended values, R9 drift test mechanics, R10 changelog repair, R11 Covenant
contact, R12 platform facts, R13 neutral examples, R14 Spec Kit branch).

## Phase 1: Design (complete)

- [data-model.md](./data-model.md): documents, sources of truth, the drift
  checks D1–D14 (plus D15–D18 added during implementation), metadata item
  fields.
- [contracts/docs.md](./contracts/docs.md): README section outline with exact
  headings and strings the tests assert; troubleshooting rows; undo rows;
  community file required content; help output contract; issue form fields;
  maintainer runbook table (PR vs admin vs web UI); link host allowlist.
- [quickstart.md](./quickstart.md): how to run the drift tests, mutate a source
  of truth to see the matching test fail, review the README as GitHub renders
  it, and verify the community profile and metadata after the maintainer acts.

Constitution re-check after design: all gates pass (table above).

## Phase 2: Tasks

Generated in [tasks.md](./tasks.md): setup, foundational tests, one phase per
user story in priority order (US1, US2, US3 at P1; US7, US4, US5 at P2; US6 at
P3), polish, maintainer actions (never executed by the PR), and an
FR-to-task/test traceability matrix.
