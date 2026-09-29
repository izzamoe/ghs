# Research: ghs Public Discoverability and Contributor Readiness

**Feature**: `002-public-discoverability` | **Date**: 2026-09-29 | **Spec**: [spec.md](./spec.md)

Every finding below was taken from the working tree at commit `dd041a4` and
from read-only queries against GitHub (`gh repo view`, `gh api`, `gh release
view`, `git ls-remote`), the Go module proxy, and pkg.go.dev on 2026-09-29.
Nothing was changed on GitHub. No unresolved clarification markers remain in
[plan.md](./plan.md).

## Part A. Audit of the current state

### A1. Repository metadata (`gh repo view --json ...`, `gh api repos/izzamoe/ghs`)

| Item | Current value | Source |
|------|---------------|--------|
| Name / visibility | `izzamoe/ghs`, public, not a fork, not archived | `gh repo view` |
| Description | empty (`""` / `null`) | `gh repo view`, `gh api` |
| Homepage | empty | same |
| Topics | none (`[]`) | `gh api repos/izzamoe/ghs/topics` |
| License | MIT (detected) | `gh repo view` |
| Primary language | Go | same |
| Default branch | `main`; no branch protection (404), no rulesets (`[]`) | `gh api .../branches/main/protection`, `.../rulesets` |
| Issues | enabled; blank issues enabled; no issue templates; no PR templates | `gh repo view` |
| Discussions | disabled | same |
| Wiki | enabled (unused) | same |
| Projects | enabled (unused) | same |
| Code of conduct | none | same |
| Security policy | none (`isSecurityPolicyEnabled: false`, `securityPolicyUrl: ""`) | same |
| Private vulnerability reporting | disabled (`{"enabled": false}`) | `gh api repos/izzamoe/ghs/private-vulnerability-reporting` |
| Secret scanning / push protection | enabled / enabled | `gh api repos/izzamoe/ghs` |
| Dependabot security updates | disabled | same |
| Funding links | none | `gh repo view` |
| Social preview | default auto-generated card (`usesCustomOpenGraphImage: false`) | same |
| Labels | GitHub's nine defaults only | `gh label list` |
| Community profile | `health_percentage: 28`; present: README, license; missing: code of conduct, contributing, issue template, PR template, description | `gh api repos/izzamoe/ghs/community/profile` |
| Viewer permission | ADMIN (the maintainer's own account was used for the audit) | `gh repo view` |
| Stars / forks / open issues / PRs | 0 / 0 / 0 / 0 (two merged PRs) | same |

### A2. Releases and tags

| Item | Finding |
|------|---------|
| Remote tags | `v0.1.0` … `v0.5.0`; `v0.5.0` is an annotated tag whose commit is `dd041a4` (re-pointed after PR #2). No `v0.5.1` tag. |
| GitHub releases | exactly one: `v0.5.0`, published 2026-09-28T13:18:50Z, not draft, not pre-release. |
| v0.5.0 assets | `checksums.txt` plus six archives: `ghs_0.5.0_{darwin,linux}_{amd64,arm64}.tar.gz`, `ghs_0.5.0_windows_{amd64,arm64}.zip`; archives bundle `LICENSE`, `README.md`, `CHANGELOG.md` (per `.goreleaser.yaml`). |
| v0.5.0 notes | body is `"\n"` (empty). Locally, `go run ./internal/tools/changelog -tag v0.5.0` prints the full section, so the extractor works; the published notes were not updated when the tag was re-pointed and the workflow re-ran. Repair requires `gh release edit` (maintainer). |
| CHANGELOG | has `## [0.5.1] - 2026-09-28` and `## [0.5.0]`; link references at the bottom stop at `0.4.0`, and `[Unreleased]` compares `v0.4.0...HEAD`. Drift. |
| Go proxy | `@latest` = `v0.5.0` (origin `refs/tags/v0.5.0`, hash `dd041a4`); `@v/list` shows all five tags. |
| pkg.go.dev | `https://pkg.go.dev/github.com/izzamoe/ghs` and `.../cmd/ghs` both return HTTP 200. |
| Workflow runs | `ci` green on `main`; `release` on `v0.5.0` failed once (24s, notes missing) then succeeded after PR #2. |

### A3. README (`README.md`, 298 lines)

Present and accurate: intro, github.com-only, Install (`go install`, Go version
from `go.mod`, PATH note, mention of prebuilt binaries, no-`sudo`), Quick start
(`import-all`, `use --fix-remote`, `doctor`), Commands block (kept in sync with
help by test), exit-code table, per-command explanations for add/use/workspace/
list/status/doctor/remotes/remove/update, Contributing paragraph (`make check`,
`make e2e`, hermetic guarantee, release procedure), License.

Missing or weak (mapped to spec stories):

| Gap | Story |
|-----|-------|
| No release-archive install steps (which file, checksum verification, where to put the binary); no explicit "no package manager" statement | US1 |
| No ordered first-run workflow that starts at `gh auth login` and ends at `doctor`; no uninstall | US1 |
| Flag help text (the `--flag  description` lines of `ghs <cmd> --help`) is not in the README; only usage lines | US2 |
| No troubleshooting or recovery section; rollback output lines (`restored origin to`, `restored gh account`) are not explained; no manual-undo table | US3 |
| Config/data privacy is scattered: config path and atomic write are stated, but identity files, SSH block format, network calls, "no tokens", "no telemetry" are not gathered | US4 |
| No cross-platform section; Windows paths and OpenSSH requirement absent | US4 |
| Examples use the maintainer's real login (`zamyb`) and full name | US4 (FR-018) |
| Contributing section does not link to a CONTRIBUTING file (none exists) | US5 |
| No links to SECURITY/SUPPORT/CODE_OF_CONDUCT (none exist) | US5 |

### A4. Help output and CLI (`internal/cli/app.go`, `usage.go`, `args.go`)

- `ghs --help` prints title, the command list, `Run "ghs <command> --help"`, and
  four footer lines (`Config:`, `Only github.com is supported.`, `Do not run ghs
  with sudo.`, `Exit codes: ...`). No pointer to the README or repository.
- `ghs <command> --help` prints `cmdSpec.usageText()`: usage lines then one
  `  --flag <value>   help` line per flag, or `(ghs <cmd> takes no flags)`.
- `ghs help <command>`: `Run` treats `args[0] == "help"` as general help and
  ignores `args[1:]`, so `ghs help use` prints the general help. Discoverability
  gap; small, testable change.
- `--version` / `-v` alias to `version` exists.
- The command table is the single source of truth; `commandListLines()` and
  `usageText()` are what the new README tests will compare against.

### A5. Existing drift tests (`internal/cli/docs_test.go`, `internal/e2e/flags_test.go`)

| Test | What it pins |
|------|--------------|
| `TestHelpStatesGitHubOnly` | help contains the github.com-only and no-sudo lines |
| `TestReadmeGoVersionMatchesGoMod` | every `Requires Go X` in README equals the `go` directive |
| `TestReadmeCommandListMatchesHelp` | first fenced block after `## Commands` equals `commandListLines()`; help contains each line |
| `TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` | README keywords (`exit code`, `sudo`, `make check`, `LICENSE`, …); LICENSE and CHANGELOG phrases |
| `TestHelpListsEveryRegisteredCommand` | help lists each command; `<cmd> --help` lists each flag name |
| `TestFlags_HelpPerCommandStdoutExit0` (e2e) | `--help` per command goes to stdout with exit 0 |

Not pinned today: flag help text in README, config path parity, install
instructions vs `go.mod` module path, release asset names vs `.goreleaser.yaml`,
`make` targets vs `Makefile`, CI gates vs docs, changelog link references,
community files, link targets/hosts.

### A6. Build, CI, release configuration

- `Makefile` targets: `fmt`, `vet`, `test`, `e2e`, `check` (all `.PHONY`).
- `.github/workflows/ci.yml`: matrix ubuntu/macos/windows; steps gofmt, `go vet ./...`, `go test -race -count=1 ./...`; `go-version-file: go.mod`.
- `.github/workflows/release.yml`: on `v*` tags; tests; notes via `internal/tools/changelog`; GoReleaser v2.
- `.goreleaser.yaml`: `goos: [linux, darwin, windows]`, `goarch: [amd64, arm64]`, `name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"`, windows `zip`, `checksum.name_template: checksums.txt`.
- No `.github/dependabot.yml`; actions pinned by major tag (`checkout@v5`, `setup-go@v6`, `goreleaser-action@v6`).
- `.gitattributes`: `* text=auto eol=lf`.
- Local toolchain during audit: `go1.27.1`; `go.mod` says `go 1.26.3`.

### A7. Spec Kit environment

- Branch `docs/public-discoverability` does not match `NNN-name`; `common.sh`
  treats `SPECIFY_FEATURE` only as the branch-name override; the feature
  directory is resolved from `SPECIFY_FEATURE_DIRECTORY` or
  `.specify/feature.json` (verified during implementation: with only
  `SPECIFY_FEATURE` set, `check-prerequisites.sh` reports "Feature directory
  not found"). All Spec Kit commands for this feature run with
  `SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability`, which also writes the git-ignored
  `.specify/feature.json`.

## Part B. Decisions

### R1. Two install paths, no package manager

- **Decision**: document exactly `go install github.com/izzamoe/ghs/cmd/ghs@latest`
  and the GitHub release archive; state "not available through Homebrew, Scoop,
  winget, apt, or the AUR". Checksum verification: `sha256sum --check
  --ignore-missing checksums.txt` (Linux), `shasum -a 256 --check
  --ignore-missing checksums.txt` (macOS), `(Get-FileHash .\ghs_<v>_windows_amd64.zip).Hash`
  compared against the line in `checksums.txt` (Windows PowerShell).
- **Rationale**: those are the only channels that exist (A2); constitution
  Principle VI forbids naming others. The proxy serves the module, so `go install`
  is verified to work.
- **Alternatives considered**: adding a Homebrew tap or GoReleaser `brews:`
  section (out of scope: would need a tap repository and maintainer commitment);
  `go run` one-liner (not an install).

### R2. README reproduces per-command help verbatim

- **Decision**: add a "Command reference" section with one fenced block per
  command whose content equals `cmdSpec.usageText()` exactly; the test iterates
  `commands` and asserts `strings.Contains(readme, "```\n"+spec.usageText()+"```")`.
  The existing summary block under `## Commands` stays.
- **Rationale**: byte-equality with the binary is the only test that cannot be
  gamed by paraphrase and needs no parsing of Markdown; it also means the README
  can be regenerated mechanically when the table changes.
- **Alternatives considered**: a Markdown table of flags (needs a parser and
  invites drift in wording); `go generate` that writes the README section (adds
  a generator and a stale-check; the assertion alone is enough for 15 commands).

### R3. `ghs help <command>` and a `Docs:` footer line

- **Decision**: in `App.Run`, when `args[0] == "help"` and `len(args) > 1`,
  rewrite to `[]string{args[1], "--help"}` and continue; unknown names then fall
  into the existing unknown-command usage error. `printHelp` gains one final
  line `Docs: https://github.com/izzamoe/ghs#readme`. Both are the only source
  changes in this feature.
- **Rationale**: `help <topic>` is the convention users try first; the README
  URL is a real, stable location (the repository itself). The change is pure
  argument rewriting with no tool calls, so it cannot violate Principle I.
- **Alternatives considered**: leaving `help <cmd>` as general help (silently
  ignores input, contrary to Principle III's "nothing is silently ignored");
  `ghs docs` command that opens a browser (new behavior with platform-specific
  side effects).

### R4. Troubleshooting as two tables: symptom → fix, and mutation → undo

- **Decision**: a symptom table with 14 rows (FR-011) and an undo table with 10
  rows (FR-012), both derived from the messages and mutation order in
  `specs/001-ghs-context-safety/contracts/cli.md` and the source (`use.go`
  rollback lines, `accounts.go` errors, `sshops/auth.go` classifications,
  `doctor.go` remedies). Manual undo commands: `gh auth switch --user <login>`,
  `git remote set-url origin <old>`, `git config [--global] --unset user.email`
  / `user.name`, `git config --global --unset --fixed-value
  'includeIf.gitdir/i:<path>/.path' <file>`, delete `gitconfig-<profile>`, edit
  `~/.ssh/config` to remove the `Host <alias>` block, `gh ssh-key delete` for an
  uploaded key (optional, user's choice), delete the config file to start over.
- **Rationale**: Principle VI requires recovery without `ghs`; every command
  above uses only `gh`, `git`, or an editor, and `ghs` prints the old values
  (`origin updated: old -> new`, `switched gh account: old -> new`) so the user
  has what the undo needs.
- **Alternatives considered**: a `ghs undo` command (new state to track; out of
  scope); pointing users at `doctor` only (doctor diagnoses, never repairs).

### R5. Config and data section content

- **Decision**: list, with source references: config file (`$XDG_CONFIG_HOME/ghs/config.conf`
  or `~/.config/ghs/config.conf`, `0600`, dir `0700`, atomic temp+rename,
  unknown keys preserved); identity files `<config dir>/gitconfig-<profile>`
  (`0600`, atomic); `~/.ssh/config` (append-only block: `Host`, `HostName
  github.com`, `User git`, `IdentityFile`, `IdentitiesOnly yes`); SSH key pair
  (`ssh-keygen -t ed25519`, never overwritten, `.pub` only ever read for upload);
  Git config via `git config` (local/global `user.name`/`user.email`;
  `--global --add includeIf...`); remotes via `git remote set-url origin`;
  network: `gh auth status`, `gh auth switch`, `gh api user`, `gh api
  user/emails`, `gh ssh-key add <pub>`, `ssh -T -o BatchMode=yes -o
  ConnectTimeout=10 git@<alias>`, `git clone`, `go install` (update). No tokens
  stored, no private key read, no telemetry, no update check.
- **Rationale**: every item is observable in the source and the 001 contract;
  the list is exhaustive so a reader can audit it against `internal/e2e` fakes.
- **Alternatives considered**: a separate `PRIVACY.md` (splits the README a
  release-archive reader has in hand; the section is short enough to inline).

### R6. Community files without inventing contacts

- **Decision**:
  - `SECURITY.md`: report through GitHub's private vulnerability reporting
    (`https://github.com/izzamoe/ghs/security/advisories/new`), which the
    maintainer enables before merge (maintainer action M1); fallback: open a
    public issue containing only "security report, please contact me" and wait
    for the maintainer to open a private channel. Supported: latest release only.
    No bounty. No email.
  - `CODE_OF_CONDUCT.md`: Contributor Covenant 2.1 verbatim; enforcement contact
    = `@izzamoe` via a GitHub issue, or the private reporting form when the
    report must stay private.
  - `SUPPORT.md`: self-help order, then a question issue; best effort, one
    maintainer, no response-time promise.
  - Issue forms (`bug_report.yml`, `feature_request.yml`) and `config.yml` with
    `blank_issues_enabled: false` and contact links to `SUPPORT.md` and the
    README troubleshooting anchor.
  - `.github/PULL_REQUEST_TEMPLATE.md` with the constitution's gates.
  - `.github/dependabot.yml` for `github-actions` weekly (the only third-party
    code the repo consumes).
  - No `FUNDING.yml`.
- **Rationale**: GitHub's community profile counts exactly these files; the
  private vulnerability form is GitHub-native and needs no email; Contributor
  Covenant explicitly allows any contact method.
- **Alternatives considered**: a security email (none exists; forbidden);
  GitHub Discussions as the support route (disabled today; optional maintainer
  action, so `SUPPORT.md` uses issues, which are enabled).

### R7. Repository metadata: what a PR can change versus what needs the maintainer

| Item | Channel | Exact action | Verification (read-only) |
|------|---------|--------------|--------------------------|
| CONTRIBUTING, SECURITY, CODE_OF_CONDUCT, SUPPORT, issue forms, PR template, dependabot config, README links, social preview *source* file | **PR** (files in the repository) | this feature's tasks | `gh api repos/izzamoe/ghs/community/profile` |
| Description | **Admin API/CLI; maintainer confirmation** | `gh repo edit izzamoe/ghs --description "<text>"` | `gh repo view --json description` |
| Homepage | Admin API/CLI | `gh repo edit izzamoe/ghs --homepage https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs` (or leave empty) | `gh repo view --json homepageUrl` |
| Topics | Admin API/CLI | `gh repo edit izzamoe/ghs --add-topic go --add-topic golang --add-topic cli --add-topic github --add-topic github-cli --add-topic git --add-topic ssh --add-topic ssh-config --add-topic multi-account --add-topic account-switcher --add-topic git-identity --add-topic developer-tools` | `gh api repos/izzamoe/ghs/topics` |
| Private vulnerability reporting | Admin API | `gh api -X PUT repos/izzamoe/ghs/private-vulnerability-reporting` | `gh api repos/izzamoe/ghs/private-vulnerability-reporting` |
| Dependabot security updates (optional) | Admin API | `gh api -X PUT repos/izzamoe/ghs/automated-security-fixes` | `gh api repos/izzamoe/ghs --jq .security_and_analysis` |
| Wiki off, Projects off (optional; both unused) | Admin CLI | `gh repo edit izzamoe/ghs --enable-wiki=false --enable-projects=false` | `gh repo view --json hasWikiEnabled,hasProjectsEnabled` |
| Discussions on (optional) | Admin CLI | `gh repo edit izzamoe/ghs --enable-discussions` | `gh repo view --json hasDiscussionsEnabled` |
| Delete branch on merge (optional) | Admin CLI | `gh repo edit izzamoe/ghs --delete-branch-on-merge` | `gh repo view --json deleteBranchOnMerge` |
| Branch protection / ruleset requiring `ci` (optional) | Admin API | ruleset via `gh api -X POST repos/izzamoe/ghs/rulesets` (body in runbook) | `gh api repos/izzamoe/ghs/rulesets` |
| v0.5.0 release notes repair | Admin CLI | `go run ./internal/tools/changelog -tag v0.5.0 -file CHANGELOG.md > /tmp/notes.md && gh release edit v0.5.0 --notes-file /tmp/notes.md` | `gh release view v0.5.0 --json body` |
| Tag `v0.5.1` (optional; changelog section exists) | Maintainer git push of a tag on `main` | `git tag -a v0.5.1 -m v0.5.1 <main-commit> && git push origin v0.5.1` | `gh release view v0.5.1 --json assets,body` |
| Social preview image | **Web UI only** (no REST/GraphQL endpoint) | Settings → General → Social preview → Upload (`docs/assets/social-preview.png` exported from the SVG at 1280×640) | `gh repo view --json usesCustomOpenGraphImage,openGraphImageUrl` |

- **Rationale**: `gh repo edit` in gh 2.101.0 exposes description, homepage,
  topics, and feature toggles (checked with `gh repo edit --help`); private
  vulnerability reporting and automated security fixes are REST `PUT`
  endpoints; the social preview has no API. All admin actions are recorded, none
  are executed by the pull request (Principle VI).
- **Alternatives considered**: a GitHub Actions workflow that applies settings
  from a file (needs a token with admin scope stored as a secret; not justified
  for a one-time change).

### R8. Recommended metadata values

- **Description** (one sentence, 158 characters): `Switch GitHub work/personal
  context safely: GitHub CLI account, Git identity, SSH host alias, and origin
  remote in one command (github.com only, no root).`
- **Homepage**: `https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs` (HTTP 200 on
  2026-09-29). Alternative: leave empty. Not: any invented site.
- **Topics** (12, lowercase, hyphenated): `go`, `golang`, `cli`, `github`,
  `github-cli`, `git`, `ssh`, `ssh-config`, `multi-account`, `account-switcher`,
  `git-identity`, `developer-tools`.
- **Social preview**: text-only SVG, 1280×640, dark text on light background,
  lines: `ghs`, the description above, `github.com/izzamoe/ghs`, `MIT`; no
  fonts beyond `sans-serif`, no external images.
- **Rationale**: description mirrors the README's first paragraph; topics are
  the terms a searcher would type; the homepage is the only third-party page
  verified to describe the module.

### R9. Drift tests: placement and mechanics

- **Decision**: keep all documentation tests in package `internal/cli`
  (`docs_test.go` extended, plus new `community_test.go` and `links_test.go`)
  because that package already reads repository files relative to `../..` and
  has access to the unexported command table. The changelog link test lives in
  `internal/tools/changelog/main_test.go` next to the extractor. CI-gate parity
  test reads `.github/workflows/ci.yml` as text and asserts the three gate
  commands documented in CONTRIBUTING appear.
- **Mechanics**:
  - Per-command reference: `strings.Contains(readme, "```\n"+spec.usageText()+"```")`.
  - Module path: regexp `^module (\S+)$` on `go.mod`; assert README contains
    `go install <module>/cmd/ghs@latest`.
  - GoReleaser: regexps `goos: \[(.*)\]`, `goarch: \[(.*)\]`,
    `name_template: "(.*)"` (first occurrence under `archives`),
    `name_template: (checksums\.txt)`; assert README contains
    `ghs_<version>_<os>_<arch>` (template rendered with the literal `<version>`
    placeholder), each `<os>_<arch>` pair, `.zip` next to `windows`, and
    `checksums.txt`.
  - Make targets: parse `.PHONY:` line of `Makefile`; regexp `` `make (\w+)` ``
    over README and CONTRIBUTING; every match must be a `.PHONY` target.
  - Changelog links: regexp `^## \[(\d+\.\d+\.\d+)\]` and
    `^\[(\d+\.\d+\.\d+)\]: https://github\.com/izzamoe/ghs/releases/tag/v\1$`;
    assert set equality; `[Unreleased]: .../compare/v<newest>...HEAD`.
  - Links: regexp `\]\(([^)\s]+)\)` over the six documents; skip `#anchors`;
    relative targets (after stripping `#fragment`) must exist via
    `os.Stat(filepath.Join("..", "..", target))`; absolute targets must parse
    with `net/url` and have `Host` in the allowlist. No network.
  - Community files: existence plus required phrase lists (see
    [contracts/docs.md](./contracts/docs.md)); `FUNDING.yml` must not exist
    under `.github/` or the root.
  - Doctor names: literal slice `{"gh", "git identity", "origin", "ssh config",
    "ssh auth", "workspace"}` in the test (the e2e doctor tests pin the same
    names), each must appear in the README between the `## Troubleshooting`
    heading and the next `## `.
  - Help parity: run `New(&out,&err).Run([]string{"--help"})`, extract the
    `Config:` and `Docs:` lines, assert README contains both paths and the URL.
- **Rationale**: pure string and regexp checks, no Markdown parser, no network,
  no new dependency (Principle IV); each test names the stale document in its
  failure message.
- **Alternatives considered**: a Markdown AST library (third-party dependency);
  a shell script in CI (would not run in `go test`, and Windows contributors
  could not run it).

### R10. Changelog repair

- **Decision**: add `[0.5.0]` and `[0.5.1]` link references, point
  `[Unreleased]` at `compare/v0.5.1...HEAD`, and add an `[Unreleased]` entry
  for this feature. The test requires the compare base to equal the newest
  *section* (`0.5.1`).
- **Implementation finding (2026-09-29)**: `https://github.com/izzamoe/ghs/releases/tag/v0.5.1`
  and `https://github.com/izzamoe/ghs/compare/v0.5.1...HEAD` return HTTP 404
  (no tag), so the two links resolve only after the maintainer tags. The
  published `v0.5.0` tag points at `dd041a4`, the commit that added the
  `0.5.1` section; tagging that same commit as `v0.5.1` publishes exactly what
  the section describes. Tagging a later `main` commit would ship this
  feature's Unreleased changes under notes that do not mention them, so M6
  names `dd041a4` explicitly and is required before merge, with a fallback
  (fold the `0.5.1` bullet into `0.5.0`, delete the heading and reference,
  compare from `v0.5.0`) if the maintainer decides not to tag.
- **Rationale**: Keep a Changelog requires the references; the drift test in R9
  makes the omission impossible to repeat.
- **Alternatives considered**: removing the `0.5.1` section unilaterally
  (rewrites history that PR #2 deliberately added; left as the maintainer's
  fallback).

### R11. Contributor Covenant contact wording

- **Decision**: "Instances of abusive, harassing, or otherwise unacceptable
  behavior may be reported to the maintainer, @izzamoe, by opening an issue in
  this repository; if the report must stay private, use GitHub's private
  vulnerability reporting form for this repository and state that it is a
  conduct report."
- **Rationale**: the Covenant's only requirement is a stated contact method;
  both routes exist without an email.
- **Alternatives considered**: omitting the contact (invalid Covenant);
  inventing an email (forbidden).

### R12. Windows and macOS notes: what is true

- Windows: `os.UserHomeDir()` is `%USERPROFILE%`; config lives at
  `%USERPROFILE%\.config\ghs\config.conf` unless `XDG_CONFIG_HOME` is set;
  SSH config at `%USERPROFILE%\.ssh\config`; `IdentityFile` written with
  `filepath.ToSlash` and quoted when it contains whitespace
  (`internal/sshops/ssh.go`); workspace and key paths accept backslashes
  (`internal/config/windows_test.go`); files are LF (`.gitattributes`, save
  tests); `ghs update` needs `go` on `PATH`; release asset is `.zip`; OpenSSH
  client is an optional Windows feature that must be installed for `doctor`'s
  `ssh auth` check and for pushes.
- macOS: bundled `git` (Command Line Tools) and OpenSSH work; `ghs` does not
  touch the keychain or `ssh-agent`; both `darwin_amd64` and `darwin_arm64`
  archives exist; Gatekeeper may quarantine a downloaded binary (`xattr -d
  com.apple.quarantine ghs` is the standard remedy; documented as a note, not a
  guarantee).
- Linux: `XDG_CONFIG_HOME` honored; nothing else platform-specific.

### R13. Neutral examples

- **Decision**: replace `zamyb`, `zamyb-work`, `IZZAMUDDIN ROYHUL FIRDAUS`, and
  `Izzam` in README examples with `alice`, `alice-work`, `Alice Example`, and
  emails at `example.com`; keep profile names `work` and `me`.
- **Rationale**: readers copy examples; the maintainer's identity is not part
  of the contract.

### R14. Spec Kit branch mismatch

- **Decision**: keep the branch name; document `SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability`
  in `quickstart.md` and `CONTRIBUTING.md`; do not create `.specify/feature.json`
  in this change.
- **Rationale**: the user asked for no git-state changes; the scripts support
  the override.
