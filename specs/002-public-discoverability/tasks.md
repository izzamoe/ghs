---

description: "Task list for ghs Public Discoverability and Contributor Readiness"
---

# Tasks: ghs Public Discoverability and Contributor Readiness

**Input**: Design documents from `/specs/002-public-discoverability/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/docs.md, quickstart.md

**Tests**: REQUIRED. Constitution Principles II and VI: the one behavior change
(`ghs help <command>`, `Docs:` line) and every documentation claim that can be
checked get a failing test first. Test tasks precede the document or code they
bind, and every task ends with a verification command.

**Organization**: Phase 1 setup, Phase 2 foundational tests, then one phase per
user story in priority order (US1, US2, US3 at P1; US7, US4, US5 at P2; US6 at
P3), polish, and maintainer actions that the pull request must not perform.
Story numbers are the stable identifiers from spec.md.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: user story label for story-phase tasks
- **[MAINTAINER]**: admin-only action; recorded with its exact command, executed by the maintainer after confirmation, never by the PR
- Every task names the exact file(s) it touches and a verification command

## Path Conventions

Single Go module at the repository root. Documentation at the root and under
`.github/` and `docs/`; documentation tests in `internal/cli/` (which already
reads `../../README.md`) and `internal/tools/changelog/`; one e2e test in
`internal/e2e/`. Spec Kit scripts run with
`SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability`.

---

## Phase 1: Setup

**Purpose**: confirm the baseline and the Spec Kit override.

- [X] T001 Record the baseline: run `make check`; build `go build -o /tmp/ghs ./cmd/ghs`; save `/tmp/ghs --help > /tmp/help-before.txt`; confirm the gap `/tmp/ghs help use` prints general help; confirm `SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability .specify/scripts/bash/check-prerequisites.sh --json` reports `FEATURE_DIR` ending in `specs/002-public-discoverability`. Files: none. **Verify**: all commands exit 0; `diff <(/tmp/ghs --help) /tmp/help-before.txt` is empty.

**Checkpoint**: baseline green; feature directory resolvable.

---

## Phase 2: Foundational (drift tests that every story's documents must satisfy)

**Purpose**: write the cross-document tests first; they fail until the documents exist.

- [X] T002 [P] Create `internal/cli/links_test.go`: `var allowedLinkHosts = []string{"github.com", "pkg.go.dev", "go.dev", "cli.github.com", "docs.github.com", "keepachangelog.com", "semver.org", "www.contributor-covenant.org"}` (`opengraph.githubassets.com` dropped: nothing references it); `docLinks(t, name string) []string` using regexp `` \]\(([^)\s]+)\) `` over `readRepoFile`; `TestDocLinksResolve` iterating `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`: skip targets starting with `#`; for `http(s)://` parse with `net/url` and require `Host` in the allowlist; otherwise strip `#fragment` and require `os.Stat(filepath.Join("..", "..", target))` to succeed; failure message names document and link. No network. **Verify**: `go test ./internal/cli -run TestDocLinksResolve` FAILS (CONTRIBUTING.md missing) — expected red.
- [X] T003 [P] Create `internal/cli/community_test.go`: `TestCommunityFilesPresentAndComplete` with the table from `contracts/docs.md` §3 (file → required substrings, forbidden substrings; `.github/FUNDING.yml` and `FUNDING.yml` must not exist; `docs/assets/social-preview.svg` must contain `viewBox="0 0 1280 640"` and must not contain `<image`, `@import`, or `href="http`; the `xmlns="http://www.w3.org/2000/svg"` declaration is allowed); `TestDocsMakeTargetsExist` parsing the `.PHONY:` line of `../../Makefile` and every `` `make (\w+)` `` in `README.md` and `CONTRIBUTING.md`; `TestContributingMatchesCIGates` asserting `gofmt -l`, `go vet ./...`, and `go test -race` appear in both `.github/workflows/ci.yml` and `CONTRIBUTING.md`. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete|TestDocsMakeTargetsExist|TestContributingMatchesCIGates'` FAILS — expected red.
- [X] T004 [P] Extend `internal/tools/changelog/main_test.go` with `TestChangelogLinkReferences`: read `../../../CHANGELOG.md`; collect versions from `^## \[(\d+\.\d+\.\d+)\]`; collect references from `^\[(\d+\.\d+\.\d+)\]: https://github\.com/izzamoe/ghs/releases/tag/v(\d+\.\d+\.\d+)$` requiring the two captures to be equal; assert the two sets are equal; assert a line `[Unreleased]: https://github.com/izzamoe/ghs/compare/v<first version heading>...HEAD` exists. **Verify**: `go test ./internal/tools/changelog -run TestChangelogLinkReferences` FAILS on the current file (missing 0.5.0 and 0.5.1 references, Unreleased base v0.4.0) — expected red.

**Checkpoint**: three red tests define the cross-document contract.

---

## Phase 3: User Story 1 — Install and first run from the README alone (Priority: P1) 🎯 MVP

**Goal**: two real install paths, prerequisites, first-run workflow, updating, uninstall; neutral examples.

**Independent Test**: on a machine without `ghs`, follow only "Install" and "First run"; end with `ghs doctor` showing no `fail`.

### Tests for User Story 1

- [X] T005 [US1] Extend `internal/cli/docs_test.go`: `TestReadmeInstallUsesModulePath` (regexp `^module (\S+)$` on `go.mod`; README must contain `go install <module>/cmd/ghs@latest`); `TestReadmeReleaseArchivesMatchGoreleaser` (parse `goos: [...]`, `goarch: [...]`, the archives `name_template`, and `checksums.txt` from `../../.goreleaser.yaml`; README must contain `ghs_<version>_<os>_<arch>.tar.gz`, `ghs_<version>_windows_<arch>.zip`, `checksums.txt`, and every `<os>_<arch>` pair); `TestReadmeUsesNeutralExamples` (README must not contain `zamyb`, `IZZAMUDDIN`, `Izzam <`); `TestReadmeSectionOrder` (the level-2 and level-3 headings from `contracts/docs.md` §2.1 appear in that order). **Verify**: `go test ./internal/cli -run 'TestReadmeInstallUsesModulePath|TestReadmeReleaseArchivesMatchGoreleaser|TestReadmeUsesNeutralExamples|TestReadmeSectionOrder'` FAILS — expected red.

### Implementation for User Story 1

- [X] T006 [US1] Rewrite the top of `README.md` through the end of `### Uninstall` per `contracts/docs.md` §2.1 and §2.3: title line, the two badges (`ci.yml/badge.svg`, `pkg.go.dev/badge`), `## Contents` (links to every level-2 heading), `## What ghs does` (current intro paragraphs, github.com-only), `## Install` with `### Requirements` (gh, git ≥ 2.30, OpenSSH; purpose of each), `### Option A: Go toolchain` (`Requires Go 1.26.3 or newer`, `go install github.com/izzamoe/ghs/cmd/ghs@latest`, PATH note for POSIX and Windows), `### Option B: Release archive` (six-row table with the literal pattern, download from `https://github.com/izzamoe/ghs/releases/latest`, checksum commands for Linux/macOS/PowerShell, extract, place on PATH, macOS quarantine note), the exact sentence `ghs is not distributed through Homebrew, Scoop, winget, apt, or the AUR; the two options above are the only supported install paths.`, `### Verify the install`, `### Do not use sudo` (existing text), `### Updating`, `### Uninstall` (what to delete, what `ghs` never deletes). **Verify**: `go test ./internal/cli -run 'TestReadmeInstallUsesModulePath|TestReadmeReleaseArchivesMatchGoreleaser|TestReadmeGoVersionMatchesGoMod'` passes; `grep -c "not distributed through Homebrew" README.md` is 1.
- [X] T007 [US1] Add `## First run` to `README.md` per `contracts/docs.md` §2.4: seven ordered steps (`gh auth login --hostname github.com` per account, `ghs import-all`, `ghs list`, `ghs init-ssh <profile> --upload`, `ghs use <profile> --fix-remote`, `ghs doctor`, optional `ghs workspace`), each with a representative output block taken from the e2e assertions (`internal/e2e/use_test.go`, `internal/e2e/doctor_test.go`, `internal/e2e/upload_test.go`). **Verify**: `grep -n "^## First run" README.md` finds one heading; `grep -c 'gh auth login --hostname github.com\|ghs import-all\|ghs init-ssh\|--fix-remote\|ghs doctor' README.md` ≥ 5; `go test ./internal/cli -run TestReadmeSectionOrder` reports the heading in order (may still fail on later headings until T012/T014/T016/T024).
- [X] T008 [US1] Replace every example identity in the retained sections of `README.md` (`## Profiles`, `## Switching`, `## Workspaces`, `## Seeing where you are`, `## Remotes and cloning`) with `alice`, `alice-work`, `Alice Example`, `alice@example.com`, `work@example.com`; rename `## Add a profile` to `## Profiles`. **Verify**: `go test ./internal/cli -run TestReadmeUsesNeutralExamples` passes; `grep -c "zamyb\|IZZAMUDDIN" README.md` is 0.

**Checkpoint**: a reader can install (either path) and complete first run from the README; tests D3, D4, D5, D14 green.

---

## Phase 4: User Story 2 — Every command and flag is discoverable from help and README (Priority: P1)

**Goal**: `ghs help <command>` works; help points at the docs; README reproduces every command's help verbatim.

**Independent Test**: for all 15 commands, README block == `ghs <cmd> --help` == `ghs help <cmd>`.

### Tests for User Story 2

- [X] T009 [US2] Extend `internal/cli/docs_test.go`: `TestHelpCommandAliasMatchesFlag` (for each `commands` entry, `Run([]string{"help", name})` stdout equals `Run([]string{name, "--help"})` stdout, both err nil); `TestHelpUnknownTopicIsUsageError` (`Run([]string{"help", "nosuch"})` returns `*UsageError` whose Message contains `unknown command "nosuch"`); `TestHelpSurplusArgumentIsUsageError` (`Run([]string{"help", "use", "--global"})` returns `*UsageError` containing `unexpected argument "--global"`); `TestHelpFooterHasDocsLine` (help output ends with `Docs: https://github.com/izzamoe/ghs#readme\n`). Extend `internal/e2e/flags_test.go`: `TestFlags_HelpCommandPerCommand` (for every `cliCommands` entry, `sb.run("help", c.name)` exit 0, stdout equals `sb.run(c.name, "--help")` stdout, zero tool calls); `TestFlags_HelpUnknownTopicExit2` (`sb.run("help", "nosuch")` exit 2, stderr starts `ghs: unknown command "nosuch"`, no tool calls). **Verify**: `go test ./internal/cli -run 'TestHelp(CommandAlias|UnknownTopic|SurplusArgument|FooterHasDocs)'` and `go test ./internal/e2e -run 'TestFlags_Help(CommandPerCommand|UnknownTopicExit2)'` FAIL — expected red.

### Implementation for User Story 2

- [X] T010 [US2] Edit `internal/cli/app.go`: in `Run`, before `lookupCommand`, add `if args[0] == "help" && len(args) > 1 { if len(args) > 2 { return usageErrorf("help", "unexpected argument %q", args[2]) }; args = []string{args[1], "--help"} }` (keep the existing `len(args)==0 || args[0]=="help" || ...` general-help branch for bare `help`); in `printHelp`, append `Docs: https://github.com/izzamoe/ghs#readme\n` after the `Exit codes:` line. No other source change. **Verify**: the T009 tests pass; `go test ./internal/cli -run 'TestHelpStatesGitHubOnly|TestHelpListsEveryRegisteredCommand'` still passes; `gofmt -l internal/cli` prints nothing.
- [X] T011 [US2] Extend `internal/cli/docs_test.go`: `TestReadmeCommandReferenceMatchesHelp` (locate the `## Command reference` section; for each `commands` entry assert the section contains "```\n" + `spec.usageText()` + "```", in table order; failure prints the expected block); `TestReadmeMatchesHelpFooter` (run help; extract the `Config:` line's two paths and the `Docs:` URL; assert each appears in README). **Verify**: `go test ./internal/cli -run 'TestReadmeCommandReferenceMatchesHelp|TestReadmeMatchesHelpFooter'` FAILS — expected red.
- [X] T012 [US2] Edit `README.md`: keep `## Commands` (summary block unchanged), `### Exit codes` (existing table plus the sentence that success output goes to stdout and errors to stderr prefixed `ghs:`), add `### Flag rules` (existing flag paragraph: any position, `--`, strict rejection), add `## Command reference` containing fifteen `### \`ghs <command>\`` subsections each with the block generated by `quickstart.md` §4 (`/tmp/ghs <command> --help`), make sure the config path sentence uses the exact text `$XDG_CONFIG_HOME/ghs/config.conf` and `~/.config/ghs/config.conf`, and add the line `Full documentation: https://github.com/izzamoe/ghs#readme` under `## What ghs does` so the `Docs:` URL from help appears in the README. **Verify**: `go test ./internal/cli -run 'TestReadmeCommandReferenceMatchesHelp|TestReadmeMatchesHelpFooter|TestReadmeCommandListMatchesHelp'` passes.

**Checkpoint**: US2 scenarios 1–5 pass; D1, D2, D6, D13 green.

---

## Phase 5: User Story 3 — A user in trouble recovers using the README (Priority: P1)

**Goal**: symptom → fix table (14 rows) and mutation → undo table (10 rows), all executable without `ghs`.

**Independent Test**: each row's manual command works against the fake-tool contract; each `doctor` check name and outcome is explained.

### Tests for User Story 3

- [X] T013 [US3] Extend `internal/cli/docs_test.go`: `TestReadmeTroubleshootingCoversDoctorChecks` (extract the text between `\n## Troubleshooting\n` and the next `\n## `; assert it contains each of `gh`, `git identity`, `origin`, `ssh config`, `ssh auth`, `workspace` and each of `ok`, `warn`, `fail`, `skip`; assert it contains the row symptoms `Permission denied (publickey)`, `Host key verification failed`, `restored origin to`, `could not restore`, `did you mean`, `is not logged in for github.com`, `not available for local builds`, `sudo`, and the undo commands `gh auth switch --user`, `git remote set-url origin`, `git config --global --unset --fixed-value`, `gh ssh-key delete`). **Verify**: `go test ./internal/cli -run TestReadmeTroubleshootingCoversDoctorChecks` FAILS — expected red.

### Implementation for User Story 3

- [X] T014 [US3] Add `## Troubleshooting` to `README.md` with an intro paragraph explaining `doctor`'s six checks and four outcomes, `### Symptoms and fixes` (the 14 rows of `contracts/docs.md` §2.5 verbatim in the symptom column), and `### What ghs changes and how to undo it` (the 10 rows of §2.6). Cross-check every `ghs` output string against `specs/001-ghs-context-safety/contracts/cli.md` and `internal/cli/use.go` (`restored origin to`, `restored gh account`, `could not restore`), `internal/cli/accounts.go` (`is not logged in for github.com`), `internal/cli/update.go` (`not available for local builds`), `internal/sshops/auth.go` (classifications). **Verify**: `go test ./internal/cli -run TestReadmeTroubleshootingCoversDoctorChecks` passes; reviewer ticks each manual command against the mutating-operation list in `specs/001-ghs-context-safety/contracts/fake-tools.md` and confirms no row deletes keys or logs out an account (FR-014).

**Checkpoint**: US3 scenarios 1–9 pass; D9 green.

---

## Phase 6: User Story 7 — Documentation cannot silently drift (Priority: P2)

**Goal**: all drift tests exist, run in CI, and each source-of-truth mutation fails exactly its test.

**Independent Test**: `quickstart.md` §3 drill.

- [X] T015 [US7] Run the mutation drill from `quickstart.md` §3 for D2–D13 (twelve mutations), restoring each file with `git checkout -- <file>`; record a table `mutation → failing test name` in the pull request description. Files touched temporarily only. **Verify**: `git status --short` after the drill shows only the intended feature changes; every row of the table names exactly the expected test.
- [X] T016 [US7] Confirm `.github/workflows/ci.yml` needs no change (it runs `go test -race -count=1 ./...`, which includes the new tests) and add a one-line comment to its `go test -race` step listing the new drift tests by file (`docs_test.go`, `community_test.go`, `links_test.go`, `changelog/main_test.go`). File: `.github/workflows/ci.yml` (comment only). **Verify**: `go test ./... -run 'Readme|Help|Community|DocLinks|DocsMake|ContributingMatches|ChangelogLink|Flags_Help' -count=1` passes; CI run on the PR is green on ubuntu, macos, windows.

**Checkpoint**: SC-005 demonstrated.

---

## Phase 7: User Story 4 — Config, data, and platform notes (Priority: P2)

**Goal**: one section that lists every file, every network call, and the "never" statements; one section per platform.

**Independent Test**: every claim traceable to source or the 001 contract.

- [X] T017 [US4] Add `## Config and data` to `README.md` per `contracts/docs.md` §2.7 with `### Files ghs writes` (rows 1–7 of the undo table with modes and write strategy), `### Files ghs reads`, `### Network access` (the exact subprocess commands and the command that triggers each), `### What ghs never does`, `### Config file format` (the example profile section, unknown keys preserved, safe to hand-edit while `ghs` is not running). Sources: `internal/config/paths.go`, `internal/config/save.go`, `internal/config/identity.go`, `internal/sshops/ssh.go`, `internal/sshops/auth.go`, `internal/ghops/user.go`, `internal/gitops/includeif.go`. **Verify**: `go test ./internal/cli -run 'TestReadmeMatchesHelpFooter|TestReadmeSectionOrder'` passes; reviewer confirms each listed subprocess appears in `specs/001-ghs-context-safety/contracts/fake-tools.md` or `internal/cli/update.go` and that no other network-reaching call exists (`grep -rn '"gh"\|"ssh"\|"git"\|"go"' internal --include='*.go' | grep -v _test`).
- [X] T018 [US4] Add `## Cross-platform notes` to `README.md` per `contracts/docs.md` §2.8 with `### Linux`, `### macOS`, `### Windows`. Sources: `internal/config/windows_test.go`, `internal/sshops/windows_test.go`, `internal/sshops/ssh.go` (`filepath.ToSlash`, `quoteIfNeeded`), `.gitattributes`, `.goreleaser.yaml` (zip on windows). **Verify**: `grep -n "^### Windows\|USERPROFILE\|Get-FileHash\|OpenSSH Client" README.md` finds each; `go test ./internal/cli -run TestReadmeSectionOrder` passes.

**Checkpoint**: US4 scenarios 1–5 pass.

---

## Phase 8: User Story 5 — Contributor, security, support, conduct, templates, runbook (Priority: P2)

**Goal**: every community health file exists with the required content; README links them; the maintainer runbook records release steps and admin-only settings.

**Independent Test**: `TestCommunityFilesPresentAndComplete`, `TestDocLinksResolve`, and after merge the community profile API.

- [X] T019 [P] [US5] Create `CONTRIBUTING.md` (contributor half): prerequisites (Go per `go.mod`, gh, git, OpenSSH, optional make), build (`go build ./cmd/ghs`), gates (`make check` = `gofmt -l .` must print nothing, `go vet ./...`, `go test -race -count=1 ./...`; `make e2e`; plain-command equivalents for Windows), the hermetic guarantee (fake `gh`/`git`/`ssh`/`ssh-keygen` in `internal/e2e`, throwaway HOME, no network), the test-first rule ("write the failing test first"), the Spec Kit sequence (`specs/<NNN-name>/`, `SPECIFY_FEATURE_DIRECTORY=specs/<NNN-name>` when the branch name differs), the `CHANGELOG.md` Unreleased rule, README/help parity (`go test ./internal/cli -run Readme`), commit and PR expectations, and the statement that there is no package-manager distribution to update. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/CONTRIBUTING|TestDocsMakeTargetsExist|TestContributingMatchesCIGates'` passes for the contributor-half substrings (the runbook substrings pass after T025).
- [X] T020 [P] [US5] Create `SECURITY.md` per `contracts/docs.md` §3: `## Supported versions` (latest release only; table with `v0.5.x` supported, older not), `## Reporting a vulnerability` (private form `https://github.com/izzamoe/ghs/security/advisories/new`; fallback: open an issue containing only `security report, please contact me` and no details; what to include; acknowledgement is best effort), `## Scope` (identity switching, key generation and upload, SSH config writes, config parsing, remote rewriting, subprocess argument handling; explicitly: `ghs` never reads private key material), `## Out of scope` (bugs in `gh`, `git`, OpenSSH, GitHub itself), `## No bug bounty` (exact phrase `no bug bounty`). No email address. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/SECURITY'` passes; `grep -E "@[a-z0-9.-]+\.[a-z]{2,}" SECURITY.md` prints nothing.
- [X] T021 [P] [US5] Create `SUPPORT.md`: self-help order (`ghs doctor --offline`, `ghs status`, `README.md#troubleshooting`), search existing issues, open a question issue via the chooser, what to include (`ghs version`, OS, redacted output), the statement `best effort` by `one maintainer` with `no response-time promise`. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/SUPPORT'` passes.
- [X] T022 [P] [US5] Create `CODE_OF_CONDUCT.md`: Contributor Covenant version 2.1 text with its attribution block and link `https://www.contributor-covenant.org`, enforcement contact per research R11 (`@izzamoe` via an issue; private reports via `https://github.com/izzamoe/ghs/security/advisories/new` marked as a conduct report). **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/CODE_OF_CONDUCT'` passes; `grep -c "INSERT CONTACT METHOD" CODE_OF_CONDUCT.md` is 0.
- [X] T023 [P] [US5] Create `.github/ISSUE_TEMPLATE/bug_report.yml` (issue form: `name: Bug report`, labels `bug`; inputs with ids `version` (`ghs version` output), `platform` (OS and architecture, `gh --version`, `git --version`), `command`, `expected`, `actual`, `doctor` (`ghs doctor --offline` output), a markdown block `Do not paste private keys, tokens, or full config files`), `.github/ISSUE_TEMPLATE/feature_request.yml` (`name: Feature request`, labels `enhancement`; ids `problem`, `proposal`, `alternatives`), `.github/ISSUE_TEMPLATE/config.yml` (`blank_issues_enabled: false`; contact links to `https://github.com/izzamoe/ghs/blob/main/SUPPORT.md` and `https://github.com/izzamoe/ghs/blob/main/README.md#troubleshooting`). **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/.github'` passes; YAML parses (`python3 -c 'import yaml,sys;[yaml.safe_load(open(f)) for f in sys.argv[1:]]' .github/ISSUE_TEMPLATE/*.yml` if PyYAML is available, otherwise `ruby -ryaml -e 'ARGV.each{|f| YAML.load_file(f)}' .github/ISSUE_TEMPLATE/*.yml`; if neither exists, the post-merge check in T033 is the verification).
- [X] T024 [P] [US5] Create `.github/PULL_REQUEST_TEMPLATE.md` (summary, linked spec or issue, checklist: `make check` passed locally; a failing test was written first for behavior changes; `CHANGELOG.md` Unreleased updated for user-visible changes; `README.md` and `--help` updated for CLI changes; no new runtime dependency; constitution Principles I–III and VI considered) and `.github/dependabot.yml` (`version: 2`, `package-ecosystem: "github-actions"`, `directory: "/"`, `schedule: interval: "weekly"`). Do NOT create `.github/FUNDING.yml`. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete'` passes for both files; `test ! -e .github/FUNDING.yml && test ! -e FUNDING.yml`.
- [X] T025 [US5] Append `## Maintainer runbook` to `CONTRIBUTING.md`: `### Cutting a release` (add the `## [x.y.z] - date` section and its link reference to `CHANGELOG.md`, move `[Unreleased]` base, merge to `main`, `git tag -a vx.y.z -m vx.y.z && git push origin vx.y.z`, watch `gh run watch`, verify `gh release view vx.y.z --json assets,body` shows seven assets and non-empty notes equal to `go run ./internal/tools/changelog -tag vx.y.z`), `### Repository settings (maintainer-only)` with the full table from `contracts/docs.md` §6 (channel, exact command, verify, required) and the ruleset JSON body for the optional `ci`-required ruleset, each row marked `maintainer-only`, and the statement that pull requests never perform these. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/CONTRIBUTING|TestDocLinksResolve'` passes; `grep -c "maintainer-only" CONTRIBUTING.md` ≥ 10.
- [X] T026 [US5] Edit `README.md`: replace the old `## Contributing` paragraph with a short `## Contributing` (gates in one line, link `[CONTRIBUTING.md](CONTRIBUTING.md)`, hermetic-suite sentence, `[CHANGELOG.md](CHANGELOG.md)`), add `## Security` (one sentence, link `[SECURITY.md](SECURITY.md)`), `## Support` (link `[SUPPORT.md](SUPPORT.md)`, mention `[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)`), keep `## License` (`[LICENSE](LICENSE)`). **Verify**: `go test ./internal/cli -run 'TestDocLinksResolve|TestReadmeSectionOrder|TestReadmeStatesExitCodesGitHubOnlyAndNoSudo'` passes.

**Checkpoint**: US5 scenarios 1–7 pass; D7, D8, D10, D11 green.

---

## Phase 9: Changelog and release documentation (US5, US7)

- [X] T027 Edit `CHANGELOG.md`: add `[0.5.0]: https://github.com/izzamoe/ghs/releases/tag/v0.5.0` and `[0.5.1]: https://github.com/izzamoe/ghs/releases/tag/v0.5.1`; change `[Unreleased]` to `https://github.com/izzamoe/ghs/compare/v0.5.1...HEAD`; under `## [Unreleased]` add `### Added` (`ghs help <command>` prints that command's help, same as `ghs <command> --help`; `ghs --help` ends with a `Docs:` line; `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`, issue forms, pull request template, Dependabot for GitHub Actions) and `### Changed` (README restructured: two install paths with checksum verification and an explicit no-package-manager statement, first-run workflow, verbatim command reference, troubleshooting with manual undo, config and data, cross-platform notes, neutral example identities; drift tests bind README, help, `go.mod`, `.goreleaser.yaml`, `Makefile`, CI, changelog, community files, and links). **Verify**: `go test ./internal/tools/changelog` passes (including `TestChangelogLinkReferences` and the existing `TestExtractSectionForTag`); `go run ./internal/tools/changelog -tag Unreleased -file CHANGELOG.md` prints the new entries; `go test ./internal/cli -run TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` still passes.

---

## Phase 10: User Story 6 — Repository metadata (Priority: P3)

**Goal**: recommendations and the social preview source are in the repository; admin-only changes are recorded, not executed.

- [X] T028 [P] [US6] Create `docs/assets/social-preview.svg`: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1280 640">` (the `xmlns` URI is the only `http` allowed; D10 forbids `href="http`, `<image`, and `@import`), light background rectangle, text lines in `font-family="sans-serif"`: `ghs` (large), `Switch GitHub work/personal context safely`, `GitHub CLI account · Git identity · SSH alias · origin remote`, `github.com/izzamoe/ghs · MIT`; no `<image>`, no external fonts. **Verify**: `go test ./internal/cli -run 'TestCommunityFilesPresentAndComplete/docs'` passes; open the file in a browser and confirm it is legible at 640×320.
- [X] T029 [US6] Confirm the metadata recommendations (description, homepage, topics, social preview) in `CONTRIBUTING.md` (T025) match research R8 exactly and that each row has a channel, an exact command or UI path, and a read-only verification. File: `CONTRIBUTING.md` (review, edit if needed). **Verify**: `grep -c "gh repo edit izzamoe/ghs" CONTRIBUTING.md` ≥ 3; `grep -c "Social preview" CONTRIBUTING.md` ≥ 1; `grep -c "no REST" CONTRIBUTING.md` ≥ 1 (or equivalent "web UI only" wording).

**Checkpoint**: US6 scenarios 1–5 satisfiable by the maintainer using only the runbook.

---

## Phase 11: Polish and cross-cutting

- [X] T030 [P] Run `make check` on Linux; run `go test -race -count=1 ./...` on macOS and Windows via CI (push the branch, open the PR). **Verify**: CI green on all three runners; the PR description contains the T015 mutation table.
- [X] T031 [P] Run the quickstart §5 rendering review on the PR and fix any broken Contents anchors in `README.md`. **Verify**: every `## ` heading has a working `#anchor` link in `## Contents` (GitHub lowercases and hyphenates; backticks are dropped).
- [X] T032 Final honesty sweep on `README.md`, `CONTRIBUTING.md`, `SECURITY.md`, `SUPPORT.md`, `CODE_OF_CONDUCT.md`: `grep -rniE "brew install|scoop install|winget install|apt install ghs|yay -S|homebrew tap"` prints nothing; `grep -rnE "@[a-z0-9.-]+\.(com|dev|org|id|net)" README.md CONTRIBUTING.md SECURITY.md SUPPORT.md CODE_OF_CONDUCT.md` prints only `example.com` addresses inside README examples; `grep -rniE "guarantee|within [0-9]+ (hours|days)|SLA"` prints nothing. **Verify**: the three greps behave as stated.
- [ ] T033 After merge to `main`: run quickstart §7 (`gh api repos/izzamoe/ghs/community/profile`) and record the result in the PR or a follow-up comment. **Verify**: `code_of_conduct`, `contributing`, `issue_template`, `pull_request_template`, `license`, `readme` all present; `gh repo view izzamoe/ghs --json isSecurityPolicyEnabled` is `true`.

---

## Phase 12: Maintainer actions (recorded here; never executed by the pull request)

Each requires the maintainer's explicit confirmation. Commands are exact; verification is read-only. See `contracts/docs.md` §6 for the full table.

- [ ] M1 [MAINTAINER] Enable private vulnerability reporting **before merging SECURITY.md**: `gh api -X PUT repos/izzamoe/ghs/private-vulnerability-reporting`. **Verify**: `gh api repos/izzamoe/ghs/private-vulnerability-reporting` → `{"enabled":true}`; `https://github.com/izzamoe/ghs/security/advisories/new` no longer 404s.
- [ ] M2 [MAINTAINER] Set the description: `gh repo edit izzamoe/ghs --description "Switch GitHub work/personal context safely: GitHub CLI account, Git identity, SSH host alias, and origin remote in one command (github.com only, no root)."`. **Verify**: `gh repo view izzamoe/ghs --json description`.
- [ ] M3 [MAINTAINER] Add topics: `gh repo edit izzamoe/ghs --add-topic go --add-topic golang --add-topic cli --add-topic github --add-topic github-cli --add-topic git --add-topic ssh --add-topic ssh-config --add-topic multi-account --add-topic account-switcher --add-topic git-identity --add-topic developer-tools`. **Verify**: `gh api repos/izzamoe/ghs/topics` lists 12 names.
- [ ] M4 [MAINTAINER] (optional) Set the homepage: `gh repo edit izzamoe/ghs --homepage https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs`, or leave empty. **Verify**: `gh repo view izzamoe/ghs --json homepageUrl`.
- [ ] M5 [MAINTAINER] Repair the v0.5.0 release notes: `go run ./internal/tools/changelog -tag v0.5.0 -file CHANGELOG.md > "${TMPDIR:-/tmp}/notes.md" && gh release edit v0.5.0 --notes-file "${TMPDIR:-/tmp}/notes.md"`. **Verify**: `gh release view v0.5.0 --json body --jq .body` is non-empty and starts with `### Added`.
- [ ] M6 [MAINTAINER] (required before merge, or apply the fallback) Tag `v0.5.1` at `dd041a4`, the commit that added the changelog's `0.5.1` section and that `v0.5.0` already points at, so the `[0.5.1]` and `[Unreleased]` links (HTTP 404 today) resolve: `git tag -a v0.5.1 -m "v0.5.1" dd041a43b17fdfce66f107c8509dc9b464dfa09a && git push origin v0.5.1`. Never tag a later `main` commit as `v0.5.1`: its Unreleased changes are not in the `0.5.1` notes. **Verify**: `gh run list --workflow release --limit 1`; `gh release view v0.5.1 --json assets,body` shows seven assets and the `0.5.1` notes. Fallback if the maintainer decides not to tag: move the `0.5.1` bullet into the `0.5.0` section's `### Fixed`, delete the `## [0.5.1]` heading and its `[0.5.1]:` reference, and point `[Unreleased]` at `compare/v0.5.0...HEAD`; `TestChangelogLinkReferences` confirms the edit.
- [ ] M7 [MAINTAINER] (recommended) Upload the social preview: export `docs/assets/social-preview.svg` to PNG at 1280×640 (any browser "save as image", `rsvg-convert -w 1280 -h 640`, or Inkscape), then Settings → General → Social preview → Upload. No API exists. **Verify**: `gh repo view izzamoe/ghs --json usesCustomOpenGraphImage` → `true`.
- [ ] M8 [MAINTAINER] (optional) `gh repo edit izzamoe/ghs --enable-wiki=false --enable-projects=false --delete-branch-on-merge`; (optional) `gh repo edit izzamoe/ghs --enable-discussions`; (optional) `gh api -X PUT repos/izzamoe/ghs/automated-security-fixes`. **Verify**: `gh repo view izzamoe/ghs --json hasWikiEnabled,hasProjectsEnabled,deleteBranchOnMerge,hasDiscussionsEnabled`; `gh api repos/izzamoe/ghs --jq .security_and_analysis`.
- [ ] M9 [MAINTAINER] (optional) Ruleset requiring the `ci` checks on `main`: `gh api -X POST repos/izzamoe/ghs/rulesets --input ruleset.json` with the body from `CONTRIBUTING.md`. **Verify**: `gh api repos/izzamoe/ghs/rulesets`.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none.
- **Foundational (Phase 2)**: after Setup; T002–T004 are parallel; they stay red until Phases 8 and 9.
- **US1 (Phase 3)**, **US2 (Phase 4)**, **US3 (Phase 5)**: all edit `README.md`, so they run sequentially in that order; T009/T010 (source change) can run in parallel with Phase 3 since they touch `internal/cli/app.go` and test files only.
- **US7 (Phase 6)**: after Phases 3–5 and 8–9 (the drill needs every test and document in place); T016 can be done any time.
- **US4 (Phase 7)**: after Phase 5 (README section order).
- **US5 (Phase 8)**: T019–T024 parallel (different files); T025 after T019; T026 after T022 (links must resolve).
- **Changelog (Phase 9)**: any time after T004; before T015.
- **US6 (Phase 10)**: T028 any time; T029 after T025.
- **Polish (Phase 11)**: after everything above; T033 after merge.
- **Maintainer actions (Phase 12)**: M1 before merging `SECURITY.md`; M6 (or its fallback) before merge; M2–M4 any time; M5 any time; M7–M9 any time.

### Within Each User Story

- Tests are written and observed failing before the document or code they bind.
- README edits are ordered by section so `TestReadmeSectionOrder` converges.

### Parallel Opportunities

- T002, T003, T004 (foundational tests).
- T009/T010 alongside T005–T008.
- T019, T020, T021, T022, T023, T024 (community files), T028 (SVG).
- T030, T031.

---

## Implementation Strategy

### MVP First (US1 + US2 + US3)

1. Phase 1, Phase 2 (red tests).
2. Phase 3: install, first run, neutral examples.
3. Phase 4: `help <command>`, `Docs:` line, command reference.
4. Phase 5: troubleshooting and undo tables.
5. **STOP and VALIDATE**: a reader can install, learn every flag, and recover; D1–D6, D9, D13, D14 green.

### Incremental Delivery

6. Phase 7 (config/data, platforms) and Phase 8 (community files) → community profile complete.
7. Phase 9 (changelog) → D12 green; Phase 10 (SVG, runbook review).
8. Phase 6 drill and Phase 11 polish → CI green on three platforms.
9. Phase 12 by the maintainer, in the order M1, M6, M5, M2, M3, M4 before merge, then M7, M8, M9.

---

## Traceability: FR → tasks and tests

| FR | Tasks | Test(s) |
|----|-------|---------|
| FR-001 | T006 | `TestReadmeInstallUsesModulePath`, `TestReadmeGoVersionMatchesGoMod` |
| FR-002 | T006 | `TestReadmeReleaseArchivesMatchGoreleaser` |
| FR-003 | T006 | `TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` (sudo), reviewer |
| FR-004 | T007 | `TestReadmeSectionOrder`, reviewer walk (quickstart §6) |
| FR-005 | T006 | `TestReadmeSectionOrder` |
| FR-006 | T011, T012 | `TestReadmeCommandReferenceMatchesHelp` |
| FR-007 | T012 | `TestReadmeCommandListMatchesHelp` (existing) |
| FR-008 | T009, T010 | `TestHelpCommandAliasMatchesFlag`, `TestHelpUnknownTopicIsUsageError`, `TestHelpSurplusArgumentIsUsageError`, `TestFlags_HelpCommandPerCommand`, `TestFlags_HelpUnknownTopicExit2` |
| FR-009 | T010, T012 | `TestHelpFooterHasDocsLine`, `TestReadmeMatchesHelpFooter` |
| FR-010 | T012 | `TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` |
| FR-011 | T014 | `TestReadmeTroubleshootingCoversDoctorChecks` (symptom strings), reviewer |
| FR-012 | T014 | `TestReadmeTroubleshootingCoversDoctorChecks` (undo commands), reviewer |
| FR-013 | T014 | `TestReadmeTroubleshootingCoversDoctorChecks` |
| FR-014 | T014 | reviewer against `contracts/fake-tools.md` (001) |
| FR-015 | T017 | `TestReadmeSectionOrder`, `TestReadmeMatchesHelpFooter`, reviewer grep |
| FR-016 | T017 | reviewer |
| FR-017 | T018 | `TestReadmeSectionOrder`, grep in T018 |
| FR-018 | T008 | `TestReadmeUsesNeutralExamples` |
| FR-019 | T019, T025 | `TestCommunityFilesPresentAndComplete/CONTRIBUTING`, `TestDocsMakeTargetsExist`, `TestContributingMatchesCIGates` |
| FR-020 | T020 | `TestCommunityFilesPresentAndComplete/SECURITY` |
| FR-021 | T022 | `TestCommunityFilesPresentAndComplete/CODE_OF_CONDUCT` |
| FR-022 | T021 | `TestCommunityFilesPresentAndComplete/SUPPORT` |
| FR-023 | T023 | `TestCommunityFilesPresentAndComplete/.github/ISSUE_TEMPLATE/*` |
| FR-024 | T024 | `TestCommunityFilesPresentAndComplete/.github/PULL_REQUEST_TEMPLATE.md` |
| FR-025 | T024 | `TestCommunityFilesPresentAndComplete` (dependabot present, FUNDING absent) |
| FR-026 | T026 | `TestDocLinksResolve`, `TestCommunityFilesPresentAndComplete` (README link substrings) |
| FR-027 | T025, T029 | grep in T029, reviewer |
| FR-028 | T025, T029 | reviewer against research R8 |
| FR-029 | T028, T025 | `TestCommunityFilesPresentAndComplete/docs/assets/social-preview.svg` |
| FR-030 | T025, M1–M9 | verification commands in Phase 12 |
| FR-031 | T011 | `TestReadmeCommandReferenceMatchesHelp` |
| FR-032 | T005 | `TestReadmeInstallUsesModulePath`, `TestReadmeGoVersionMatchesGoMod` |
| FR-033 | T005 | `TestReadmeReleaseArchivesMatchGoreleaser` |
| FR-034 | T003 | `TestDocsMakeTargetsExist` |
| FR-035 | T004 | `TestChangelogLinkReferences` |
| FR-036 | T002 | `TestDocLinksResolve` |
| FR-037 | T003 | `TestCommunityFilesPresentAndComplete` |
| FR-038 | T011, T013 | `TestReadmeMatchesHelpFooter`, `TestReadmeTroubleshootingCoversDoctorChecks` |
| FR-039 | T003 | `TestContributingMatchesCIGates` |
| FR-040 | T002–T005, T009, T011, T013, T016 | all of the above run under `go test ./...` with no network |
| FR-041 | T027 | `TestChangelogLinkReferences`, `TestExtractSectionForTag` |
| FR-042 | T025, M5, M6 | `gh release view --json body` in the runbook |

## Notes

- Write each test, run it, watch it fail, then write the document or code (constitution Principle II).
- `[P]` tasks touch different files and have no dependency on an incomplete task.
- Commit after each task or logical group; never commit with `make check` failing; never commit a mutated file from the T015 drill.
- The pull request must not run any `[MAINTAINER]` command; those need the maintainer's confirmation and admin rights.
- Nothing in this feature invents a package manager, website, contact address, funding link, or guarantee; if a reviewer finds one, it is a defect.

## Implementation notes (2026-09-29)

Executed on branch `docs/public-discoverability`. T001–T032 are done; T033 and
M1–M9 are left for after review (T033 needs the merge; M1–M9 need admin
rights and were not run). Evidence is in the pull request description.

Test-first: every test in T002–T005, T009, T011, and T013 was written and run
before the code or document it binds; the red run failed exactly these 20
tests: `TestChangelogLinkReferences`, `TestCommunityFilesPresentAndComplete`,
`TestContributingMatchesCIGates`, `TestDocLinksResolve`,
`TestDocsMakeNoInventedClaims`, `TestDocsMakeTargetsExist`,
`TestDocURLHostsAllowed`, `TestFlags_HelpCommandPerCommand`,
`TestFlags_HelpUnknownTopicExit2`, `TestHelpCommandAliasMatchesFlag`,
`TestHelpFooterHasDocsLine`, `TestHelpSurplusArgumentIsUsageError`,
`TestHelpUnknownTopicIsUsageError`, `TestReadmeCommandReferenceMatchesHelp`,
`TestReadmeInstallUsesModulePath`, `TestReadmeMatchesHelpFooter`,
`TestReadmeReleaseArchivesMatchGoreleaser`, `TestReadmeSectionOrder`,
`TestReadmeTroubleshootingCoversDoctorChecks`, `TestReadmeUsesNeutralExamples`.

Deviations from the planned text, each because the plan was wrong or
unverifiable, not to reduce scope:

- Spec Kit override: `SPECIFY_FEATURE` alone does not resolve the feature
  directory with the installed scripts ("Feature directory not found");
  `SPECIFY_FEATURE_DIRECTORY=specs/<NNN-name>` does. All artifacts,
  `CONTRIBUTING.md`, and the community test use it.
- `v0.5.1`: the plan assumed the `compare/v0.5.1...HEAD` link would show
  "nothing to compare"; it returns HTTP 404, as does the `v0.5.1` release
  link. M6 now tags `dd041a4` (not a later `main`) before merge, or applies
  the fallback.
- Exact messages taken from the source instead of the contract draft: the
  config error is `invalid config <path>: line <n>: ...`; a failed rollback
  names the old URL and login in its `could not restore ...` lines (no
  success lines are printed on failure); `import-all` prints
  `imported <n> profiles from gh host "github.com"`.
- README examples in First run were captured by running the real binary
  against the hermetic fakes with the neutral identities (a throwaway test,
  deleted afterwards), not typed by hand.
- Added troubleshooting row: key upload failing on a missing
  `admin:public_key` scope (gh's minimum scopes are `repo`, `read:org`,
  `gist`).
- macOS checksum: `shasum -a 256 <archive>` compared with `checksums.txt`,
  because `--ignore-missing` support in macOS `shasum` was not verified.
- `ghs help help`, `ghs help --help`, and `ghs help -h` print the general
  help (the rewrite runs before the general-help check).
- Extra drift tests beyond the plan: `TestDocURLHostsAllowed` (split from the
  link test; also scans code blocks and `.github` templates),
  `TestHeadingAnchor` and anchor checking in `TestDocLinksResolve`,
  `TestDocsMakeNoInventedClaims`, `TestSecuritySupportedVersionMatchesChangelog`.
- The README documents an existing behavior instead of changing it (out of
  scope): `ghs update` on a release-archive install would `go install` a
  second copy into `$(go env GOPATH)/bin` and leave the running binary
  unchanged, so archive users are told to download the next archive.
