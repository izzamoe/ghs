# Contributing to ghs

Thanks for helping. `ghs` switches identities and writes files that people
rely on every day, so the rules below are strict on purpose.

`ghs` has one maintainer ([@izzamoe](https://github.com/izzamoe)). Reviews are
best effort.

## Prerequisites

- Go at the version in the `go` directive of [go.mod](go.mod) or newer.
- Git 2.30 or newer, the [GitHub CLI](https://cli.github.com) (`gh`), and an
  OpenSSH client, if you want to try the binary by hand. The test suite does
  not need any of them: it uses fakes.
- `make` is optional. Every target below has a plain-command equivalent, so
  Windows contributors without `make` can run the same gates.

## Build

```bash
go build -o ghs ./cmd/ghs    # Windows: go build -o ghs.exe ./cmd/ghs
./ghs --help
```

A local build reports `ghs devel` from `ghs version`, and `ghs update` refuses
to run on it.

## Gates

Pull requests must pass the same gates CI runs on Linux, macOS, and Windows
(`.github/workflows/ci.yml`):

```bash
make check   # all three gates below, in order
make fmt     # gofmt -l . must print nothing
make vet     # go vet ./...
make test    # go test -race -count=1 ./...
make e2e     # only the end-to-end suite in internal/e2e
```

Without `make`:

```bash
gofmt -l .                        # must print nothing
go vet ./...
go test -race -count=1 ./...
go test -race -count=1 ./internal/e2e/...
```

The race detector needs cgo; if your platform cannot build with `-race`, run
`go test -count=1 ./...` locally and let CI run the race build.

## The test suite is hermetic

The suite runs without network access, without a GitHub account, without SSH
keys, and without touching your real `$HOME`, `~/.ssh`, `~/.gitconfig`, or
GitHub CLI state:

- Unit tests live next to the code in `internal/...`.
- End-to-end tests in `internal/e2e` build the real `ghs` binary and run it
  against fake `gh`, `git`, `ssh`, and `ssh-keygen` executables placed on
  `PATH`, in a throwaway home directory with a minimal environment. The fakes
  record every invocation and answer from scripted state, so a test can assert
  the exact order of tool calls, rollbacks, and exit codes.
- Never replace a fake with a real account, a real key, or a network call. If
  a behavior cannot be tested with the fakes, extend the fakes.

## Write the failing test first

Every behavior change starts with a failing test, then the change that makes
it pass (Red, Green, Refactor). For a defect, the pull request should contain
a test that fails without the fix. Documentation claims that can be checked
mechanically are tests too; see "Documentation drift tests" below.

## Non-trivial changes start with a short design note

For a change that touches more than one command, alters user-visible
behavior, or restructures documentation, describe the problem, the proposed
behavior, and the plan for tests in the pull request description (or a linked
issue) before writing code. There is no required template file for this; the
failing-test rule above and the pull request template are the only
enforced gates.

## Changelog

Every change that alters user-visible behavior (commands, flags, output, exit
codes, files written, documentation a user follows) adds a line under
`## [Unreleased]` in [CHANGELOG.md](CHANGELOG.md), in
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) form. While the major
version is `0`, mark breaking CLI changes with `**Breaking:**`.

## Help text and README stay identical

The command table in `internal/cli/app.go` is the single source of truth for
usage lines and flag help. `README.md` reproduces `ghs <command> --help` byte
for byte under "Command reference". After changing the table, regenerate that
section:

```bash
go build -o /tmp/ghs ./cmd/ghs
for c in add-profile add-from-gh import-all list status doctor set-email workspace use clone init-ssh fix-remote remove version update; do
  printf '### `ghs %s`\n\n```\n' "$c"; /tmp/ghs "$c" --help; printf '```\n\n'
done
```

and paste the output into `README.md`. `go test ./internal/cli -run Readme`
tells you what is still out of date.

## Documentation drift tests

These run inside `go test ./...`, need no network, and fail with the name of
the stale document:

| Test | Keeps in step |
|------|---------------|
| `TestReadmeCommandListMatchesHelp`, `TestReadmeCommandReferenceMatchesHelp` | README command list and reference with `ghs --help` and `ghs <command> --help` |
| `TestHelpCommandAliasMatchesFlag`, `TestFlags_HelpCommandPerCommand` | `ghs help <command>` with `ghs <command> --help` |
| `TestReadmeGoVersionMatchesGoMod`, `TestReadmeInstallUsesModulePath` | README install instructions with `go.mod` |
| `TestReadmeReleaseArchivesMatchGoreleaser` | README archive names with `.goreleaser.yaml` |
| `TestReadmeMatchesHelpFooter` | README config path and docs URL with the help footer |
| `TestReadmeTroubleshootingCoversDoctorChecks` | README troubleshooting with the six `doctor` checks and recovery commands |
| `TestReadmeSectionOrder`, `TestReadmeUsesNeutralExamples` | README outline and placeholder identities |
| `TestDocsMakeTargetsExist` | every `make <target>` mentioned with the `Makefile` |
| `TestContributingMatchesCIGates` | this file and the `Makefile` with the CI gate commands |
| `TestCommunityFilesPresentAndComplete`, `TestDocsMakeNoInventedClaims` | community files, no funding file, no invented contact, channel, or promise |
| `TestDocLinksResolve`, `TestDocURLHostsAllowed` | relative links and anchors resolve; absolute links use allowed hosts |
| `TestChangelogLinkReferences` | changelog headings with their link references |

The allowed hosts for absolute links are listed in `allowedLinkHosts` in
`internal/cli/links_test.go`. Adding one is a reviewed change.

## Honest documentation

Do not document an install channel, website, contact address, funding
account, or promise that does not exist. `ghs` is installed only with
`go install` or from a GitHub release archive; there is no package-manager
distribution to update when you release. Where a reader would expect one of
these, say plainly that it does not exist.

## Commits and pull requests

- Use Conventional Commits style subjects, as the history does: `feat:`, `fix:`, `docs:`, `test:`,
  `ci:`, `chore:`, with an optional scope, for example
  `fix(cli): reject empty --ssh-key`.
- Keep a pull request to one concern. Fill in the pull request template: it
  asks for `make check`, the failing test, the changelog line, README and help
  updates, and confirmation that no runtime dependency was added.
- `ghs` uses the standard library only. A new third-party runtime dependency
  needs a recorded justification in the pull request description.
- Security problems are never discussed in public issues or pull requests;
  see [SECURITY.md](SECURITY.md).
- Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).

## Maintainer runbook

Everything in this section is **maintainer-only**. Contributors and pull
requests never perform these steps; a pull request can only change files in
the repository.

### Cutting a release

1. On a branch, move the `## [Unreleased]` entries of `CHANGELOG.md` into a
   new `## [X.Y.Z] - YYYY-MM-DD` section, add the reference
   `[X.Y.Z]: https://github.com/izzamoe/ghs/releases/tag/vX.Y.Z`, and change
   the `[Unreleased]` reference to
   `https://github.com/izzamoe/ghs/compare/vX.Y.Z...HEAD`.
   `go test ./internal/tools/changelog` checks all three.
2. Check the notes the release will publish:
   `go run ./internal/tools/changelog -tag vX.Y.Z -file CHANGELOG.md`.
3. Merge the pull request to `main` with CI green.
4. Tag the merge commit and push the tag (maintainer-only):

   ```bash
   git fetch origin
   git tag -a vX.Y.Z -m vX.Y.Z origin/main
   git push origin vX.Y.Z
   ```

5. The `release` workflow (`.github/workflows/release.yml`) runs the test
   suite, extracts the notes from the changelog section, and GoReleaser
   publishes six archives and `checksums.txt`. Watch it with
   `gh run list --workflow release --limit 1` and `gh run watch <run id>`.
6. Verify the release (read-only):

   ```bash
   gh release view vX.Y.Z --json assets --jq '.assets[].name'   # six archives and checksums.txt
   gh release view vX.Y.Z --json body --jq .body                # must not be empty
   diff <(gh release view vX.Y.Z --json body --jq .body) <(go run ./internal/tools/changelog -tag vX.Y.Z -file CHANGELOG.md)
   go list -m github.com/izzamoe/ghs@vX.Y.Z                      # the module proxy serves the tag
   ```

   If the notes are empty or differ, repair them with
   `gh release edit vX.Y.Z --notes-file <file>` (see the table below).
7. Never move a tag that has been pushed. The Go module proxy caches the
   first commit it sees for a version; fix a bad release with a new patch
   version.

### Repository settings (maintainer-only)

Some repository metadata can only be changed by an administrator through the
API, the GitHub CLI, or the web UI. Each row gives the channel, the exact
command, and a read-only verification. Run a row only after deciding to;
pull requests never perform them.

| Item | Channel | Command or path | Verify (read-only) | Required | Who |
|------|---------|-----------------|--------------------|----------|-----|
| Community files, templates, Dependabot config, social preview source | pull request | files in this repository | `gh api repos/izzamoe/ghs/community/profile --jq .files` | yes | contributors, reviewed by the maintainer |
| Private vulnerability reporting | admin API | `gh api -X PUT repos/izzamoe/ghs/private-vulnerability-reporting` | `gh api repos/izzamoe/ghs/private-vulnerability-reporting` prints `{"enabled":true}` | yes, before `SECURITY.md` is relied on | maintainer-only |
| Description | admin CLI | `gh repo edit izzamoe/ghs --description "Switch GitHub work/personal context safely: GitHub CLI account, Git identity, SSH host alias, and origin remote in one command (github.com only, no root)."` | `gh repo view izzamoe/ghs --json description` | yes | maintainer-only |
| Topics | admin CLI | `gh repo edit izzamoe/ghs --add-topic go --add-topic golang --add-topic cli --add-topic github --add-topic github-cli --add-topic git --add-topic ssh --add-topic ssh-config --add-topic multi-account --add-topic account-switcher --add-topic git-identity --add-topic developer-tools` | `gh api repos/izzamoe/ghs/topics` | yes | maintainer-only |
| Homepage | admin CLI | `gh repo edit izzamoe/ghs --homepage https://pkg.go.dev/github.com/izzamoe/ghs/cmd/ghs`, or leave it empty; there is no project website | `gh repo view izzamoe/ghs --json homepageUrl` | optional | maintainer-only |
| Repair the empty v0.5.0 release notes | admin CLI | `go run ./internal/tools/changelog -tag v0.5.0 -file CHANGELOG.md > "${TMPDIR:-/tmp}/notes.md" && gh release edit v0.5.0 --notes-file "${TMPDIR:-/tmp}/notes.md"` | `gh release view v0.5.0 --json body --jq .body` is not empty | yes | maintainer-only |
| Tag v0.5.1 | maintainer git | `git tag -a v0.5.1 -m v0.5.1 dd041a43b17fdfce66f107c8509dc9b464dfa09a && git push origin v0.5.1` (see the note below) | `gh release view v0.5.1 --json assets,body` | yes, or apply the fallback below | maintainer-only |
| Dependabot security updates | admin API | `gh api -X PUT repos/izzamoe/ghs/automated-security-fixes` | `gh api repos/izzamoe/ghs/automated-security-fixes` | optional | maintainer-only |
| Wiki and Projects off (both unused) | admin CLI | `gh repo edit izzamoe/ghs --enable-wiki=false --enable-projects=false` | `gh repo view izzamoe/ghs --json hasWikiEnabled,hasProjectsEnabled` | optional | maintainer-only |
| Discussions on | admin CLI | `gh repo edit izzamoe/ghs --enable-discussions` | `gh repo view izzamoe/ghs --json hasDiscussionsEnabled` | optional | maintainer-only |
| Delete branch on merge | admin CLI | `gh repo edit izzamoe/ghs --delete-branch-on-merge` | `gh repo view izzamoe/ghs --json deleteBranchOnMerge` | optional | maintainer-only |
| Ruleset requiring CI on `main` | admin API | `gh api -X POST repos/izzamoe/ghs/rulesets --input ruleset.json` with the body below | `gh api repos/izzamoe/ghs/rulesets` | optional | maintainer-only |
| Social preview image | web UI only (no REST or GraphQL API exists) | Settings → General → Social preview → Upload an image exported from `docs/assets/social-preview.svg` at 1280×640 | `gh repo view izzamoe/ghs --json usesCustomOpenGraphImage` prints `true` | recommended | maintainer-only |

**About v0.5.1.** `CHANGELOG.md` has a `0.5.1` section, added by commit
`dd041a4`, but no `v0.5.1` tag exists, so the `[0.5.1]` and `[Unreleased]`
changelog links do not resolve yet. The published `v0.5.0` tag already points
at `dd041a4`, so tagging that same commit as `v0.5.1` publishes exactly what
the section describes and nothing more (the binaries are the same code as
`v0.5.0`; `ghs update` would report `v0.5.0 → v0.5.1`). Do not tag a later
`main` commit as `v0.5.1`: its Unreleased changes are not in the `0.5.1`
notes. If you decide not to tag, apply the fallback instead: move the `0.5.1`
bullet into the `0.5.0` section's `### Fixed` list, delete the `## [0.5.1]`
heading and its `[0.5.1]:` reference, and point `[Unreleased]` at
`compare/v0.5.0...HEAD`. `go test ./internal/tools/changelog` confirms either
choice.

**Social preview.** Export the SVG to PNG at 1280×640 with any browser's
"save as image", `rsvg-convert -w 1280 -h 640 docs/assets/social-preview.svg -o social-preview.png`,
or Inkscape, then upload it in the web UI. The PNG is not committed.

**Ruleset body** (`ruleset.json`) requiring the three CI jobs on the default
branch and forbidding force pushes and deletion:

```json
{
  "name": "main",
  "target": "branch",
  "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    {
      "type": "required_status_checks",
      "parameters": {
        "strict_required_status_checks_policy": false,
        "required_status_checks": [
          { "context": "test (ubuntu-latest)" },
          { "context": "test (macos-latest)" },
          { "context": "test (windows-latest)" }
        ]
      }
    }
  ]
}
```

After the maintainer actions, the community profile can be checked with:

```bash
gh api repos/izzamoe/ghs/community/profile --jq '{health: .health_percentage, files: (.files | with_entries(.value |= (. != null)))}'
gh repo view izzamoe/ghs --json isSecurityPolicyEnabled,securityPolicyUrl,issueTemplates,pullRequestTemplates
```
