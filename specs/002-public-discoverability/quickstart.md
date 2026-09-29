# Quickstart: Validating Public Discoverability and Contributor Readiness

**Feature**: `002-public-discoverability` | **Date**: 2026-09-29 | **Spec**: [spec.md](./spec.md)

All commands run from the repository root. Nothing here needs network access
except the sections marked "after merge" and "maintainer".

## 0. Spec Kit environment

The branch is `docs/public-discoverability`, which does not match the
`NNN-name` pattern the Spec Kit scripts expect. Export the override once per
shell:

```bash
export SPECIFY_FEATURE_DIRECTORY=specs/002-public-discoverability
.specify/scripts/bash/check-prerequisites.sh --json --include-tasks
# FEATURE_DIR must end with specs/002-public-discoverability
```

## 1. Baseline (before any change)

```bash
make check                                  # gofmt, go vet, go test -race ./...
go build -o /tmp/ghs ./cmd/ghs && /tmp/ghs --help && /tmp/ghs use --help
/tmp/ghs help use; echo "exit=$?"           # today: prints general help (gap, FR-008)
go test ./internal/cli -run 'Readme|Help' -v
```

## 2. Run the drift tests

```bash
go test ./internal/cli -run 'TestReadme|TestHelp|TestCommunity|TestDocs|TestContributing|TestDocLinks' -v
go test ./internal/tools/changelog -run TestChangelogLinkReferences -v
go test ./internal/e2e -run 'TestFlags_Help' -v
```

Expected on the delivered tree: all pass. Expected on the tree before the
feature: `TestChangelogLinkReferences`, `TestCommunityFilesPresentAndComplete`,
`TestDocLinksResolve`, `TestReadmeCommandReferenceMatchesHelp`,
`TestReadmeMatchesHelpFooter`, `TestReadmeReleaseArchivesMatchGoreleaser`,
`TestReadmeInstallUsesModulePath`, `TestReadmeTroubleshootingCoversDoctorChecks`,
`TestReadmeUsesNeutralExamples`, `TestHelpCommandAliasMatchesFlag` fail.

## 3. Mutation drill (SC-005): each source of truth breaks exactly its test

Run each block, confirm the named test fails, then restore the file with
`git checkout -- <file>` (or `git stash`/`git stash pop` for the whole set).
Never commit a mutated file.

| Drift class | Mutation | Test that must fail |
|-------------|----------|---------------------|
| D2 command reference | in `internal/cli/app.go` change the `--global` help text by one character | `TestReadmeCommandReferenceMatchesHelp` |
| D3 Go version | in `go.mod` change `go 1.26.3` to `go 1.26.4` | `TestReadmeGoVersionMatchesGoMod` |
| D4 module path | in README change `cmd/ghs@latest` to `cmd/ghs@main` | `TestReadmeInstallUsesModulePath` |
| D5 release assets | in `.goreleaser.yaml` add `freebsd` to `goos` | `TestReadmeReleaseArchivesMatchGoreleaser` |
| D6 help footer | in `internal/cli/app.go` change the `Docs:` URL | `TestReadmeMatchesHelpFooter` |
| D7 make targets | in `CONTRIBUTING.md` write `` `make lint` `` | `TestDocsMakeTargetsExist` |
| D8 CI gates | in `.github/workflows/ci.yml` change `go test -race` to `go test` | `TestContributingMatchesCIGates` |
| D9 doctor names | in README rename `ssh auth` to `ssh login` inside Troubleshooting | `TestReadmeTroubleshootingCoversDoctorChecks` |
| D10 community files | `mv SECURITY.md /tmp/` (move it back afterwards) | `TestCommunityFilesPresentAndComplete`; also create an empty `.github/FUNDING.yml`, confirm the same test fails, then delete it |
| D11 links | in README add `[x](https://example.com)` | `TestDocLinksResolve` |
| D12 changelog | delete the `[0.5.0]:` reference line | `TestChangelogLinkReferences` |
| D13 help alias | revert the `help <command>` rewrite in `App.Run` | `TestHelpCommandAliasMatchesFlag`, `TestFlags_HelpCommandPerCommand` |

Restore everything and confirm `git status --short` shows only intended changes.

## 4. Generate the README command reference blocks

The reference must equal `ghs <command> --help` byte for byte. Generate it
rather than typing it:

```bash
go build -o /tmp/ghs ./cmd/ghs
for c in add-profile add-from-gh import-all list status doctor set-email workspace use clone init-ssh fix-remote remove version update; do
  printf '### `ghs %s`\n\n```\n' "$c"; /tmp/ghs "$c" --help; printf '```\n\n'
done > /tmp/command-reference.md
```

Paste `/tmp/command-reference.md` under `## Command reference`. Windows
(PowerShell):

```powershell
go build -o $env:TEMP\ghs.exe .\cmd\ghs
$cmds = 'add-profile','add-from-gh','import-all','list','status','doctor','set-email','workspace','use','clone','init-ssh','fix-remote','remove','version','update'
$cmds | ForEach-Object { "### ``ghs $_``"; ''; '```'; & $env:TEMP\ghs.exe $_ --help; '```'; '' } | Set-Content -Encoding utf8 $env:TEMP\command-reference.md
```

## 5. Review the README the way readers see it

- GitHub: open the pull request, "Files changed", click "View file" on
  `README.md` for the rendered view; check the Contents anchors resolve.
- Release archive: `tar -tzf` a release archive lists `README.md`; the install
  and troubleshooting sections must make sense without `CONTRIBUTING.md`
  (which is not bundled).
- pkg.go.dev: after a release, `https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs`
  renders the README; relative links resolve against the repository root.

## 6. Walk the first-run and troubleshooting sections in the sandbox

The hermetic suite already exercises every command the README shows. To walk
the README text against the fakes without a real account:

```bash
go test ./internal/e2e -run 'TestUse_|TestDoctor_|TestFixRemote_|TestInitSSH_|TestWorkspace_' -v
```

For each troubleshooting row, find the test that produces the row's symptom
(for example `TestDoctor_SSHAuthVariants` for `Host key verification failed`,
`TestUse_FixRemoteRollbackRestoresOriginAndAccount` for `restored origin to`)
and confirm the README's wording matches the output the test asserts.

On a real machine (read-only commands only): `ghs list`, `ghs status`,
`ghs doctor --offline`, `ghs help use`, `ghs --help | tail -5`.

## 7. After merge: community profile

```bash
gh api repos/izzamoe/ghs/community/profile --jq '{health: .health_percentage, files: (.files | with_entries(.value |= (. != null)))}'
```

Expected: `code_of_conduct`, `contributing`, `issue_template`,
`pull_request_template`, `license`, `readme` all `true`; `health` reaches 100
once the maintainer sets the description (M2). Also:

```bash
gh repo view izzamoe/ghs --json isSecurityPolicyEnabled,securityPolicyUrl,issueTemplates,pullRequestTemplates
```

## 8. Maintainer actions (not performed by the pull request)

Run only with the maintainer's confirmation; each is idempotent and verifiable.
The full table with optional items is in `CONTRIBUTING.md` (Maintainer
runbook) and [contracts/docs.md](./contracts/docs.md) §6.

```bash
# M1 enable private vulnerability reporting (do this before merging SECURITY.md)
gh api -X PUT repos/izzamoe/ghs/private-vulnerability-reporting
gh api repos/izzamoe/ghs/private-vulnerability-reporting          # {"enabled":true}

# M2 description, M3 topics, M4 homepage (optional)
gh repo edit izzamoe/ghs --description "Switch GitHub work/personal context safely: GitHub CLI account, Git identity, SSH host alias, and origin remote in one command (github.com only, no root)."
gh repo edit izzamoe/ghs --add-topic go --add-topic golang --add-topic cli --add-topic github --add-topic github-cli --add-topic git --add-topic ssh --add-topic ssh-config --add-topic multi-account --add-topic account-switcher --add-topic git-identity --add-topic developer-tools
gh repo edit izzamoe/ghs --homepage https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs
gh repo view izzamoe/ghs --json description,homepageUrl,repositoryTopics

# M5 repair v0.5.0 release notes
go run ./internal/tools/changelog -tag v0.5.0 -file CHANGELOG.md > "${TMPDIR:-/tmp}/notes.md"
gh release edit v0.5.0 --notes-file "${TMPDIR:-/tmp}/notes.md"
gh release view v0.5.0 --json body --jq .body | head -3

# M6 tag v0.5.1 at dd041a4, the commit that added its changelog section (required
# before merge, or apply the fallback in CONTRIBUTING.md); never tag a later main commit
git tag -a v0.5.1 -m "v0.5.1" dd041a43b17fdfce66f107c8509dc9b464dfa09a && git push origin v0.5.1
gh run watch && gh release view v0.5.1 --json assets,body

# M7 social preview: web UI only
# Settings → General → Social preview → Upload docs/assets/social-preview.png (export the SVG at 1280×640)
gh repo view izzamoe/ghs --json usesCustomOpenGraphImage
```

## 9. Done criteria

- `make check` green locally and CI green on Linux, macOS, Windows.
- Section 3 drill: every row fails exactly its test and passes after restore.
- Section 7: every profile file present.
- `grep -rn "NEEDS[ ]CLARIFICATION:" specs/002-public-discoverability` prints nothing (the form Spec Kit uses for an open question).
- No `FUNDING.yml`, no email address, no package-manager instruction anywhere
  in the documentation set: `grep -rniE "brew install|scoop install|winget install|@[a-z0-9-]+\.(com|dev|org|id)" README.md CONTRIBUTING.md SECURITY.md SUPPORT.md CODE_OF_CONDUCT.md` prints only `example.com` sample emails inside README examples.
