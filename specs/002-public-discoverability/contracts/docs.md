# Contract: Documentation, Help Output, Community Files, and Metadata Runbook

**Feature**: `002-public-discoverability` | **Date**: 2026-09-29 | **Spec**: [spec.md](../spec.md)

This contract fixes the strings that tests assert and the structure reviewers
check. Tests reference this file; the README and community files must satisfy
it byte for byte where "exact" is stated.

## 1. Help output contract (`internal/cli/app.go`)

### 1.1 General help (`ghs`, `ghs help`, `ghs --help`, `ghs -h`)

Unchanged except one appended line. Exact footer after the command list and
the `Run "ghs <command> --help"` line:

```
Config: $XDG_CONFIG_HOME/ghs/config.conf, or ~/.config/ghs/config.conf
Only github.com is supported.
Do not run ghs with sudo.
Exit codes: 0 success, 1 failure, 2 usage error.
Docs: https://github.com/izzamoe/ghs#readme
```

### 1.2 `ghs help <command>`

| Invocation | Behavior | Exit |
|------------|----------|------|
| `ghs help use` | stdout equals `ghs use --help` output byte for byte; no tool invoked | 0 |
| `ghs help <any registered command>` | same rule for every command in the table | 0 |
| `ghs help nosuch` | stderr `ghs: unknown command "nosuch"` followed by the general usage block; stdout empty | 2 |
| `ghs help use --global` | stderr `ghs: unexpected argument "--global"` followed by the general usage block; stdout empty; no tool invoked | 2 |
| `ghs help` | general help (unchanged) | 0 |
| `ghs help help`, `ghs help --help`, `ghs help -h` | general help (asking for help about help) | 0 |

Implementation note (R3; as implemented, the rewrite runs before the general-help check so that `help help`, `help --help`, and `help -h` reach it): in `App.Run`, before the command lookup: `if args[0] == "help" && len(args) > 1 { if len(args) > 2 { return usageErrorf("help", "unexpected argument %q", args[2]) }; args = []string{args[1], "--help"} }`. `parseArgs` then reports `--help` for a registered command, and the existing unknown-command path handles an unknown topic. `usageFor("help")` falls back to the general usage block because `help` is not a registered command.

## 2. README contract (`README.md`)

### 2.1 Section order and exact headings

Level-2 headings, in this order; tests locate sections by these exact strings.

```
# ghs
## Contents
## What ghs does
## Install
### Requirements
### Option A: Go toolchain
### Option B: Release archive
### Verify the install
### Do not use sudo
### Updating
### Uninstall
## First run
## Commands
### Exit codes
### Flag rules
## Command reference
## Profiles
## Switching: `use`
## Workspaces
## Seeing where you are: `list`, `status`, `doctor`
## Remotes and cloning
## Removing a profile
## Troubleshooting
### Symptoms and fixes
### What ghs changes and how to undo it
## Config and data
### Files ghs writes
### Files ghs reads
### Network access
### What ghs never does
### Config file format
## Cross-platform notes
### Linux
### macOS
### Windows
## Contributing
## Security
## Support
## License
```

The existing sections `Profiles` (renamed from "Add a profile"), `Switching`,
`Workspaces`, `Seeing where you are`, `Remotes and cloning`, `Removing a
profile` keep their current content with neutral examples (R13).

### 2.2 Exact strings tests assert

| Test | Exact string(s) |
|------|-----------------|
| D3 (existing) | `Requires Go 1.26.3` (value read from `go.mod`) |
| D4 | `go install github.com/izzamoe/ghs/cmd/ghs@latest` (module path read from `go.mod`) |
| D5 | `ghs_<version>_<os>_<arch>.tar.gz`, `ghs_<version>_windows_<arch>.zip`, `checksums.txt`, and each pair `linux_amd64`, `linux_arm64`, `darwin_amd64`, `darwin_arm64`, `windows_amd64`, `windows_arm64` (pairs derived from `.goreleaser.yaml`) |
| D6 | `$XDG_CONFIG_HOME/ghs/config.conf`, `~/.config/ghs/config.conf`, `https://github.com/izzamoe/ghs#readme` (all read from help output) |
| D1 (existing) | first fenced block after `## Commands` equals the command list |
| D2 | for each command: ` ```\n` + `usageText()` + ` ``` ` appears inside `## Command reference` |
| D9 | inside `## Troubleshooting`: `gh`, `git identity`, `origin`, `ssh config`, `ssh auth`, `workspace`, and `ok`, `warn`, `fail`, `skip` |
| D14 | README does not contain `zamyb`, `IZZAMUDDIN`, or `Izzam <` |
| existing | `exit code`, `Only github.com is supported`, `sudo`, `` `0` ``, `` `1` ``, `` `2` ``, `make check`, `LICENSE` |
| D10 (links) | README contains `](CONTRIBUTING.md)`, `](SECURITY.md)`, `](SUPPORT.md)`, `](CODE_OF_CONDUCT.md)`, `](CHANGELOG.md)`, `](LICENSE)` |
| D7 | every `` `make <target>` `` mention is one of `fmt`, `vet`, `test`, `e2e`, `check` |

### 2.3 Install section content (Option A / Option B)

- Option A: the `go install` command; `Requires Go 1.26.3 or newer`; PATH note
  (`export PATH="$PATH:$(go env GOPATH)/bin"`); Windows equivalent
  (`%USERPROFILE%\go\bin` on PATH).
- Option B: table of the six archives with the literal pattern; steps:
  download archive and `checksums.txt` from
  `https://github.com/izzamoe/ghs/releases/latest`; verify (Linux `sha256sum
  --check --ignore-missing checksums.txt`; macOS `shasum -a 256 <archive>` compared
  with the archive's line in `checksums.txt` (as implemented: the
  `--ignore-missing` option of macOS `shasum` was not verified, so the README
  uses the comparison that works everywhere); Windows `Get-FileHash .\ghs_<version>_windows_amd64.zip
  -Algorithm SHA256` and compare with the line in `checksums.txt`); extract
  (`tar -xzf` / Expand-Archive); move `ghs` (or `ghs.exe`) to a directory on
  `PATH`; macOS note about Gatekeeper quarantine
  (`xattr -d com.apple.quarantine ghs`), stated as "may be needed".
- The sentence, exact: `ghs is not distributed through Homebrew, Scoop, winget,
  apt, or the AUR; the two options above are the only supported install paths.`
- Verify: `ghs version` (prints `ghs v0.5.0` style; local builds print `ghs
  devel`), `ghs --help`.
- Do not use sudo: existing paragraph.
- Updating: `ghs update` only for Option A installs; Option B users download the
  next archive; `ghs version` before and after.
- Uninstall: delete the binary (`$(go env GOPATH)/bin/ghs` or wherever it was
  placed); delete the config directory (`$XDG_CONFIG_HOME/ghs` or
  `~/.config/ghs`, which also removes `gitconfig-<profile>` files); then the
  exact list of what remains and is left to the user: SSH keys, `Host` blocks
  in `~/.ssh/config`, global `includeIf` entries (`git config --global
  --get-regexp includeIf` to list; `--unset` to remove), local/global
  `user.name`/`user.email`, remotes, GitHub CLI logins.

### 2.4 First run (ordered)

1. `gh auth login --hostname github.com` for each account; `gh auth status`.
2. `ghs import-all`; expected output `imported <n> profiles from gh host
   "github.com"` (the previously active account is restored silently on
   success); profiles are named after the login.
3. `ghs list`.
4. `ghs init-ssh <profile> --upload` for each profile; expected lines
   `generated ssh key`, `appended Host`, `uploaded public key` / `already registered`.
5. In a repository: `ghs use <profile> --fix-remote`; expected `origin updated`,
   `switched gh account`, `git identity set`.
6. `ghs doctor`; expected all `ok`/`skip`; mention `--offline`.
7. Optional: `ghs workspace <profile> <dir>` for directory-wide identity.

### 2.5 Troubleshooting rows (exact symptom column)

| # | Symptom | Cause | ghs fix | Manual fix |
|---|---------|-------|---------|------------|
| 1 | `git push` says `Permission denied (publickey)` | origin uses `github.com` or an alias whose key is not on the account | `ghs doctor <profile>`; then `ghs init-ssh <profile> --upload` and/or `ghs fix-remote <profile>` | `ssh -T git@<alias>`; add the `.pub` key at github.com/settings/keys; `git remote set-url origin git@<alias>:owner/repo.git` |
| 2 | `doctor` shows `ssh auth: Host key verification failed` | `doctor` runs `ssh` in batch mode and never writes `known_hosts` | none (by design) | run `ssh -T git@<alias>` once and accept GitHub's host key |
| 3 | push or PR went to the wrong account | active `gh` account, Git identity, and `origin` disagree | `ghs status`, then `ghs use <profile> --fix-remote` | `gh auth switch --user <login>`; `git remote set-url origin ...` |
| 4 | last commit has the wrong author email | identity was set for another profile or not at all | `ghs use <profile>` (or `ghs workspace`), then `git commit --amend --reset-author` | `git config user.email <email>` then amend |
| 5 | `ghs use` printed `restored origin to ...` / `restored gh account ...` | a later step failed; earlier steps were rolled back | read the first `ghs:` line for the cause; rerun after fixing it | nothing to undo |
| 6 | `ghs use` printed `could not restore ...` | rollback itself failed | fix the cause, then run the manual commands | `git remote set-url origin <url>`; `gh auth switch --user <login>`, with the URL and login named in the `could not restore origin to <url>` / `could not restore gh account <login>` lines (a failed `use` prints no success lines, so there is no `origin updated` line to read) |
| 7 | `ghs: invalid config <path>: line <n>: ...` (or `cannot read config <path>: ...`) | the config file cannot be parsed; ghs refuses to overwrite it | none (by design) | fix the named line in an editor, or move the file aside and re-import |
| 8 | `profile "Work" not found; did you mean "work"?` | names are exact; lookup suggests the case variant | use the suggested name | none |
| 9 | `git identity unchanged: profile "x" has no email` | GitHub returned no email | `ghs set-email <profile> <email>`; or `gh auth refresh --scopes user:email` then re-import | edit `git_email` in the config file |
| 10 | `gh account <login> is not logged in for github.com` | the GitHub CLI has no valid token for that login | `gh auth login --hostname github.com` | same |
| 11 | `origin ... is not a GitHub remote` or `unknown alias` | ghs rewrites only github.com URLs and known aliases | `ghs fix-remote` refuses by design | `git remote set-url origin <url>` yourself |
| 12 | `update is not available for local builds` | binary was built from source, not `go install` | `go install github.com/izzamoe/ghs/cmd/ghs@latest` | same |
| 13 | ran `ghs` with `sudo` | files were created under `/root` | none | remove `/root/.config/ghs`, `/root/.ssh/config` block, `/root/.gitconfig` entries, and `sudo gh auth logout` for accounts logged in as root; then redo first run as your user |
| 14 | Windows: `ssh is not installed` or `ssh-keygen` not found | OpenSSH client feature missing | none | install "OpenSSH Client" (Settings → Apps → Optional features) or Git for Windows' OpenSSH, and make sure it is on `PATH` |
| 15 | `init-ssh --upload` / `clone --upload-key` fails with a `gh ssh-key add` scope error | the token lacks a key scope (gh's minimum scopes are `repo`, `read:org`, `gist`) | `gh auth switch --user <login>`, `gh auth refresh --hostname github.com --scopes admin:public_key`, retry | add the `.pub` key at github.com/settings/keys |

### 2.6 Undo rows (exact "What changes" column)

| # | What changes | Commands that change it | How it is written | Manual undo |
|---|--------------|-------------------------|-------------------|-------------|
| 1 | `config.conf` | add-profile, add-from-gh, import-all, set-email, workspace, remove | atomic (temp file + rename), `0600` | edit or delete the file; `ghs` never overwrites an unreadable one |
| 2 | `gitconfig-<profile>` next to the config | workspace, set-email, add-from-gh, import-all | atomic, `0600` | delete the file after removing its `includeIf` entry |
| 3 | `~/.ssh/config` | init-ssh, clone | append one `Host <alias>` block; never rewritten | delete the block in an editor |
| 4 | SSH key pair `<key>` and `<key>.pub` | init-ssh, clone | `ssh-keygen -t ed25519`; never overwritten or deleted | delete both files yourself if unwanted |
| 5 | global `includeIf.gitdir/i:<dir>/.path` | workspace, add-profile/add-from-gh `--workspace` | `git config --global --add` | `ghs workspace <profile> --unlink`, or `git config --global --unset --fixed-value 'includeIf.gitdir/i:<dir>/.path' <file>` |
| 6 | repository or global `user.name` / `user.email` | use, clone | `git config [--global]` | `git config [--global] --unset user.name` / `user.email` |
| 7 | `origin` URL | use `--fix-remote`, fix-remote | `git remote set-url origin` | `git remote set-url origin <old>` (old URL is printed as `origin updated: <old> -> <new>`) |
| 8 | active GitHub CLI account | use, clone, import-all (temporarily), init-ssh `--upload` (temporarily) | `gh auth switch` | `gh auth switch --user <login>` |
| 9 | public key on the GitHub account | init-ssh `--upload`, clone `--upload-key` | `gh ssh-key add <key>.pub --title ghs-<profile>` | `gh ssh-key list` then `gh ssh-key delete <id>` |
| 10 | new directory from `clone` | clone | `git clone` through the alias | delete the directory |

### 2.7 Config and data (facts to state)

- Writes: rows 1–7 above with modes (`0600` files, `0700` directories).
- Reads: config file; `~/.ssh/config`; `<key>.pub` only (for upload); Git config
  via `git config --show-origin --show-scope`; `gh auth status --json hosts`.
- Network (through subprocesses only; `ghs` opens no sockets itself): `gh auth
  status`, `gh auth switch`, `gh api user`, `gh api user/emails`, `gh ssh-key
  add`, `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` (doctor,
  skipped with `--offline`), `git clone`, `go install` (update).
- Never: reads or prints private key material; stores tokens (the GitHub CLI
  does); sends telemetry; checks for updates; runs a shell; edits
  `~/.gitconfig` by hand; deletes keys or logs accounts out.
- Format example (exact keys as written by `config.Save`):

```
[work]
gh_user = "alice-work"
git_name = "Alice Example"
git_email = "alice@example.com"
ssh_host_alias = "github-work"
ssh_key = "~/.ssh/id_ed25519_work"
workspace = "~/Documents/work"
```

### 2.8 Cross-platform notes (facts to state)

- Linux: `XDG_CONFIG_HOME` honored; otherwise `~/.config/ghs`.
- macOS: bundled `git` and OpenSSH work; no keychain or `ssh-agent`
  integration; Intel and Apple Silicon archives; Gatekeeper note.
- Windows: config `%USERPROFILE%\.config\ghs\config.conf` (or
  `XDG_CONFIG_HOME`); SSH config `%USERPROFILE%\.ssh\config`; keys
  `%USERPROFILE%\.ssh\`; backslash paths accepted and normalized;
  `IdentityFile` written with forward slashes and quoted when it contains
  spaces; LF line endings; `.zip` archive; PowerShell checksum command; OpenSSH
  client required; `ghs update` needs `go` on `PATH`.

## 3. Community files: required content (asserted by D10)

| File | Must contain (substring) | Must not contain |
|------|--------------------------|------------------|
| `CONTRIBUTING.md` | `make check`, `gofmt -l`, `go vet ./...`, `go test -race`, `make e2e`, `internal/e2e`, `hermetic`, `failing test`, `specs/`, `SPECIFY_FEATURE_DIRECTORY=`, `CHANGELOG.md`, `## Maintainer runbook`, `gh repo edit`, `private-vulnerability-reporting`, `gh release edit`, `gh release view`, `--json body`, `maintainer-only` | `@example.com` email of a person; `brew`, `scoop`, `winget` as install instructions |
| `SECURITY.md` | `## Supported versions`, `latest release`, `## Reporting a vulnerability`, `https://github.com/izzamoe/ghs/security/advisories/new`, `security report, please contact me`, `## Scope`, `private key`, `## Out of scope`, `no bug bounty`; and ``latest release (currently `v<major>.<minor>.x`)`` for the newest changelog section (`TestSecuritySupportedVersionMatchesChangelog`) | `@` followed by a domain (no email) |
| `SUPPORT.md` | `ghs doctor --offline`, `ghs status`, `README.md#troubleshooting`, `search`, `best effort`, `one maintainer`, `no response-time` | `@` followed by a domain |
| `CODE_OF_CONDUCT.md` | `Contributor Covenant`, `version 2.1`, `@izzamoe`, `security/advisories/new`, `https://www.contributor-covenant.org` | `[INSERT CONTACT METHOD]`, `@` followed by a domain |
| `.github/ISSUE_TEMPLATE/bug_report.yml` | `name: Bug report`, `id: version`, `id: platform`, `id: command`, `id: expected`, `id: actual`, `id: doctor`, `ghs doctor --offline`, `Do not paste private keys` | |
| `.github/ISSUE_TEMPLATE/feature_request.yml` | `name: Feature request`, `id: problem`, `id: proposal`, `id: alternatives` | |
| `.github/ISSUE_TEMPLATE/config.yml` | `blank_issues_enabled: false`, `SUPPORT.md`, `README.md#troubleshooting` | |
| `.github/PULL_REQUEST_TEMPLATE.md` | `make check`, `failing test`, `CHANGELOG.md`, `README.md`, `--help`, `no new runtime dependency` | |
| `.github/dependabot.yml` | `package-ecosystem: "github-actions"`, `interval: "weekly"` | |
| `.github/FUNDING.yml`, `FUNDING.yml` | (must not exist) | |
| `docs/assets/social-preview.svg` | `viewBox="0 0 1280 640"`, `ghs`, `github.com/izzamoe/ghs` | `<image`, `@import`, `href="http` (the `xmlns="http://www.w3.org/2000/svg"` declaration is allowed) |

## 4. Link contract (asserted by D11)

Documents scanned: `README.md`, `CONTRIBUTING.md`, `SECURITY.md`,
`SUPPORT.md`, `CODE_OF_CONDUCT.md`, `CHANGELOG.md`.

Allowed absolute-link hosts (`allowedLinkHosts` in `internal/cli/links_test.go`):
`github.com`, `pkg.go.dev`, `go.dev`, `cli.github.com`, `docs.github.com`,
`keepachangelog.com`, `semver.org`, `www.contributor-covenant.org`.
(`opengraph.githubassets.com` was dropped during implementation because no
document references it.) Every `http(s)://` URL in the six documents and in
`.github/PULL_REQUEST_TEMPLATE.md` and `.github/ISSUE_TEMPLATE/*.yml`,
including inside code blocks, must be `https` on an allowed host
(`TestDocURLHostsAllowed`).

Relative links (`TestDocLinksResolve`, outside fenced code): the target file
must exist after stripping `#fragment`; a fragment, alone (`#troubleshooting`)
or after a Markdown file (`README.md#troubleshooting`), must match a heading
anchor generated the way GitHub does (lower case, punctuation and backticks
dropped, spaces to hyphens).

Badges permitted in the README (both hosts are allowed and both URLs exist):
`https://github.com/izzamoe/ghs/actions/workflows/ci.yml/badge.svg` and
`https://pkg.go.dev/badge/github.com/izzamoe/ghs.svg` linking to
`https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs`.

## 5. Changelog contract (asserted by D12)

- Every `## [x.y.z]` heading has exactly one
  `[x.y.z]: https://github.com/izzamoe/ghs/releases/tag/vx.y.z` line.
- `[Unreleased]: https://github.com/izzamoe/ghs/compare/v<newest section>...HEAD`.
- `## [Unreleased]` gains, under `### Added`: `ghs help <command>` and the
  `Docs:` help line; community health files; under `### Changed`: README
  restructure (install paths, first run, command reference, troubleshooting,
  config and data, cross-platform notes), neutral examples.

## 6. Maintainer runbook table (copied into `CONTRIBUTING.md`)

| Item | Channel | Command / path | Verify | Required |
|------|---------|----------------|--------|----------|
| Community files, templates, Dependabot, social preview source | PR | this feature | `gh api repos/izzamoe/ghs/community/profile --jq .files` | yes |
| Description | admin CLI (maintainer confirmation) | `gh repo edit izzamoe/ghs --description "Switch GitHub work/personal context safely: GitHub CLI account, Git identity, SSH host alias, and origin remote in one command (github.com only, no root)."` | `gh repo view izzamoe/ghs --json description` | yes |
| Topics | admin CLI | `gh repo edit izzamoe/ghs --add-topic go --add-topic golang --add-topic cli --add-topic github --add-topic github-cli --add-topic git --add-topic ssh --add-topic ssh-config --add-topic multi-account --add-topic account-switcher --add-topic git-identity --add-topic developer-tools` | `gh api repos/izzamoe/ghs/topics` | yes |
| Homepage | admin CLI | `gh repo edit izzamoe/ghs --homepage https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs` (or leave empty) | `gh repo view izzamoe/ghs --json homepageUrl` | optional |
| Private vulnerability reporting | admin API | `gh api -X PUT repos/izzamoe/ghs/private-vulnerability-reporting` | `gh api repos/izzamoe/ghs/private-vulnerability-reporting` → `{"enabled":true}` | yes, before merging SECURITY.md |
| Repair v0.5.0 release notes | admin CLI | `go run ./internal/tools/changelog -tag v0.5.0 -file CHANGELOG.md > "$TMPDIR/notes.md" && gh release edit v0.5.0 --notes-file "$TMPDIR/notes.md"` | `gh release view v0.5.0 --json body --jq .body` non-empty | yes |
| Tag v0.5.1 | maintainer git | `git tag -a v0.5.1 -m v0.5.1 dd041a43b17fdfce66f107c8509dc9b464dfa09a && git push origin v0.5.1` (the commit that added the `0.5.1` section, which `v0.5.0` also points at; never a later `main` commit) | `gh release view v0.5.1 --json assets,body` | yes before merge, or the fallback in `CONTRIBUTING.md` |
| Dependabot security updates | admin API | `gh api -X PUT repos/izzamoe/ghs/automated-security-fixes` | `gh api repos/izzamoe/ghs --jq .security_and_analysis` | optional |
| Wiki and Projects off | admin CLI | `gh repo edit izzamoe/ghs --enable-wiki=false --enable-projects=false` | `gh repo view izzamoe/ghs --json hasWikiEnabled,hasProjectsEnabled` | optional |
| Discussions on | admin CLI | `gh repo edit izzamoe/ghs --enable-discussions` | `gh repo view izzamoe/ghs --json hasDiscussionsEnabled` | optional |
| Delete branch on merge | admin CLI | `gh repo edit izzamoe/ghs --delete-branch-on-merge` | `gh repo view izzamoe/ghs --json deleteBranchOnMerge` | optional |
| Ruleset requiring `ci` on `main` | admin API | `gh api -X POST repos/izzamoe/ghs/rulesets --input ruleset.json` (body in CONTRIBUTING) | `gh api repos/izzamoe/ghs/rulesets` | optional |
| Social preview image | web UI only | Settings → General → Social preview → Upload `docs/assets/social-preview.png` (export the SVG at 1280×640) | `gh repo view izzamoe/ghs --json usesCustomOpenGraphImage` → `true` | recommended |
