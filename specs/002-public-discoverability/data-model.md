# Data Model: Documentation Sources of Truth and Drift Checks

**Feature**: `002-public-discoverability` | **Date**: 2026-09-29 | **Spec**: [spec.md](./spec.md)

This feature stores no runtime data. Its "entities" are documents, the
machine-readable sources they must agree with, and the tests that bind them.

## 1. Documents

| Document | Path | Reader | Bundled in release archive |
|----------|------|--------|----------------------------|
| README | `README.md` | user, contributor, pkg.go.dev, release archive | yes |
| Changelog | `CHANGELOG.md` | user, release workflow | yes |
| License | `LICENSE` | everyone | yes |
| Contributing guide | `CONTRIBUTING.md` | contributor, maintainer | no |
| Security policy | `SECURITY.md` | reporter | no |
| Support guide | `SUPPORT.md` | user | no |
| Code of conduct | `CODE_OF_CONDUCT.md` | community | no |
| Bug issue form | `.github/ISSUE_TEMPLATE/bug_report.yml` | user | no |
| Feature issue form | `.github/ISSUE_TEMPLATE/feature_request.yml` | user | no |
| Issue chooser config | `.github/ISSUE_TEMPLATE/config.yml` | user | no |
| PR template | `.github/PULL_REQUEST_TEMPLATE.md` | contributor | no |
| Dependabot config | `.github/dependabot.yml` | GitHub | no |
| Social preview source | `docs/assets/social-preview.svg` | maintainer | no |

## 2. Sources of truth

| Source | Path | Fact it owns |
|--------|------|--------------|
| Command table | `internal/cli/app.go` (`commands`) | command list, usage lines, flag names and help text |
| Help footer | `internal/cli/app.go` (`printHelp`) | config path text, `Docs:` URL, github.com-only, no-sudo, exit codes |
| Module file | `go.mod` | module path, minimum Go version |
| Release config | `.goreleaser.yaml` | OS/arch matrix, archive name template, zip-on-windows, checksums file name, bundled files |
| Makefile | `Makefile` (`.PHONY:`) | gate target names |
| CI workflow | `.github/workflows/ci.yml` | gate commands and platforms |
| Doctor checks | `internal/cli/doctor.go` (six literal names; pinned by `internal/e2e/doctor_test.go`) | check names and outcomes |
| Changelog headings | `CHANGELOG.md` `## [x.y.z]` | released versions needing link references |
| Filesystem | repository tree | targets of relative links |
| Host allowlist | `internal/cli/links_test.go` | permitted absolute-link hosts |

## 3. Drift checks (document × source)

| ID | Test (package) | Document | Source | Fails when |
|----|----------------|----------|--------|------------|
| D1 | `TestReadmeCommandListMatchesHelp` (cli, existing) | README `## Commands` block | command table | any usage line differs |
| D2 | `TestReadmeCommandReferenceMatchesHelp` (cli, new) | README `## Command reference` blocks | `cmdSpec.usageText()` | any command's block ≠ `--help` output |
| D3 | `TestReadmeGoVersionMatchesGoMod` (cli, existing) | README `Requires Go X` | `go.mod` go directive | version differs |
| D4 | `TestReadmeInstallUsesModulePath` (cli, new) | README install command | `go.mod` module line | `go install <module>/cmd/ghs@latest` absent |
| D5 | `TestReadmeReleaseArchivesMatchGoreleaser` (cli, new) | README release-archive section | `.goreleaser.yaml` | a pair, the template, `.zip`, or `checksums.txt` missing |
| D6 | `TestReadmeMatchesHelpFooter` (cli, new) | README | help `Config:` and `Docs:` lines | path or URL missing from README |
| D7 | `TestDocsMakeTargetsExist` (cli, new) | README, CONTRIBUTING | Makefile `.PHONY` | `make <x>` mentioned but not a target |
| D8 | `TestContributingMatchesCIGates` (cli, new) | CONTRIBUTING | `ci.yml` | `gofmt -l`, `go vet ./...`, or `go test -race` missing from either |
| D9 | `TestReadmeTroubleshootingCoversDoctorChecks` (cli, new) | README `## Troubleshooting` | six doctor names, four outcomes | any missing |
| D10 | `TestCommunityFilesPresentAndComplete` (cli, new) | all community files | required phrase lists ([contracts/docs.md](./contracts/docs.md) §3) | file missing, phrase missing, or `FUNDING.yml` present |
| D11 | `TestDocLinksResolve` (cli, new) | six Markdown documents | filesystem, host allowlist | relative target missing or host not allowed |
| D12 | `TestChangelogLinkReferences` (changelog tool, new) | CHANGELOG | its own headings | reference missing/extra or `[Unreleased]` base ≠ newest section |
| D13 | `TestHelpCommandAliasMatchesFlag` (cli, new) + `TestFlags_HelpCommandPerCommand` (e2e, new) | help behavior | command table | `ghs help <cmd>` ≠ `ghs <cmd> --help` |
| D14 | `TestReadmeUsesNeutralExamples` (cli, new) | README | literal list of forbidden strings | maintainer's real login or name appears |
| D15 | `TestReadmeSectionOrder` (cli, new) | README headings and Contents | [contracts/docs.md](./contracts/docs.md) §2.1 | a heading is missing, out of order, or not linked from Contents |
| D16 | `TestDocURLHostsAllowed` (cli, new; split from D11) | six documents plus `.github` templates, code blocks included | host allowlist | a URL is not `https` or its host is not allowed |
| D17 | `TestDocsMakeNoInventedClaims` (cli, new) | six documents plus `.github` templates | forbidden package-manager commands, promises, non-placeholder emails | an invented channel, promise, or contact appears |
| D18 | `TestSecuritySupportedVersionMatchesChangelog` (cli, new) | `SECURITY.md` | newest `CHANGELOG.md` section | the supported-version row names another minor version |

## 4. Metadata items

| Field | Type | Values |
|-------|------|--------|
| `name` | string | description, homepage, topics, wiki, projects, discussions, private-vulnerability-reporting, dependabot-security-updates, delete-branch-on-merge, ruleset, release-notes-v0.5.0, tag-v0.5.1, social-preview |
| `channel` | enum | `pr` (files), `admin-cli` (needs maintainer confirmation), `web-ui` (no API) |
| `recommended` | string | see research R8 |
| `command` | string | exact `gh` / `git` command or UI path |
| `verify` | string | read-only command |
| `required` | bool | true for description, topics, private vulnerability reporting, release notes repair; false for the rest |

The full instance table lives in research R7 and is copied into
`CONTRIBUTING.md`'s maintainer runbook by task T025.

## 5. Troubleshooting rows (README)

Each row has: `symptom` (exact message or observable), `cause`, `ghs fix`,
`manual fix`, `platform note` (optional). The 14 required rows are listed in
spec FR-011; the 10 undo rows in FR-012. Both are reproduced with final wording
in [contracts/docs.md](./contracts/docs.md) §2.5 and §2.6.
