# Quickstart: validating ghs Account-Context Safety

**Feature**: `001-ghs-context-safety` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

This guide proves the feature works end to end. Formats and exit codes are
defined in [contracts/cli.md](./contracts/cli.md); fake tool behavior in
[contracts/fake-tools.md](./contracts/fake-tools.md).

## Prerequisites

- Go at the version in `go.mod` (`go 1.26.3`); `go version` must print at least that.
- Git 2.30 or newer on `PATH` (for the real-machine smoke test only).
- The GitHub CLI (`gh`) and an OpenSSH client (for the real-machine smoke test only).
- No network is needed for the automated suite.

## 1. Build and run the local quality gates

```bash
make check          # gofmt -l (must print nothing), go vet ./..., go test -race ./...
```

Equivalent commands without `make` (Windows PowerShell works the same):

```bash
gofmt -l . && go vet ./... && go test -race ./...
```

Expected: no formatting output, `ok` for every package including
`internal/e2e`, total wall time under five minutes.

## 2. Run only the end-to-end suite

```bash
make e2e            # go test -race -count=1 ./internal/e2e/...
go test -race -run 'TestDoctor|TestWorkspace' -v ./internal/e2e/   # a subset
```

Each end-to-end test creates its own sandbox (temporary `HOME`,
`XDG_CONFIG_HOME`, and a `bin/` directory holding the fakes) and passes an
environment containing only that `bin/` on `PATH`, so nothing on the machine is
read or written. To confirm isolation, run the suite with your real home
directory read-only or with `HOME` pointed at an empty directory; it must pass
either way.

## 3. Drive the binary by hand against the fakes

The test binary is also the fake tool set, so you can script scenarios
interactively without a GitHub account:

```bash
tmp=$(mktemp -d)
go build -o "$tmp/ghs" ./cmd/ghs
go test -c -o "$tmp/e2e.test" ./internal/e2e
mkdir -p "$tmp/bin" "$tmp/home" "$tmp/cfg"
for t in gh git ssh ssh-keygen; do cp "$tmp/e2e.test" "$tmp/bin/$t"; done   # add .exe on Windows
cat > "$tmp/state.json" <<'JSON'
{"gh":{"accounts":[{"login":"zamyb-work","active":true,"state":"success"},{"login":"zamyb","active":false,"state":"success"}],
       "users":{"zamyb-work":{"id":101,"login":"zamyb-work","name":"Izzam","email":"work@example.com"}}},
 "git":{"repo":true,"toplevel":"/tmp/repo","origin":"git@github.com:acme/app.git","local":{},"global":{}},
 "ssh":{"aliases":{"github-work":{"login":"zamyb-work"}}}}
JSON
run() { env -i PATH="$tmp/bin" HOME="$tmp/home" USERPROFILE="$tmp/home" XDG_CONFIG_HOME="$tmp/cfg" \
        GHS_FAKE_STATE="$tmp/state.json" GHS_FAKE_LOG="$tmp/calls.log" "$tmp/ghs" "$@"; echo "exit=$?"; }

run add-profile work --gh-user zamyb-work --git-name Izzam --git-email work@example.com \
    --ssh-alias github-work --ssh-key '~/.ssh/id_ed25519_work'
run list                      # "* work ... active" and "active gh account: zamyb-work (profile "work")"
run use work                  # exit=0 plus a stderr warning: origin still uses github.com
run use work --fix-remote     # "origin updated: git@github.com:acme/app.git -> git@github-work:acme/app.git"
run init-ssh work             # creates fake keys under $tmp/home/.ssh and appends the Host block
run doctor                    # six lines, "summary: ... 0 fail ...", exit=0
run workspace work '~/Documents/work'
run doctor                    # workspace line is now "ok"
run remove work               # "kept:" lines for the key, the SSH block, and the gh login; exit=0
cat "$tmp/calls.log"          # every fake invocation with args and working directory
```

Expected outputs are the exact strings in [contracts/cli.md](./contracts/cli.md).

## 4. Story-by-story validation map

| Story | Automated proof (`go test -run`) | Manual check from section 3 |
|-------|----------------------------------|-----------------------------|
| US1 config never lost | `TestConfig_ ./internal/e2e/` and `TestSave ./internal/config/` | corrupt a line in `$tmp/cfg/ghs/config.conf`, run `add-profile`, file unchanged, exit 1 |
| US2 validation | `TestValidation_ ./internal/e2e/` and `TestValidate ./internal/config/` | `run add-profile 'bad name' ...` → exit 2, no file |
| US3 SSH config | `TestInitSSH_ ./internal/e2e/` and `TestEnsureConfig ./internal/sshops/` | set `HOME` to a path with a space; `IdentityFile` is quoted |
| US4 upload account | `TestInitSSHUpload_ ./internal/e2e/` | `run init-ssh work --upload` with `zamyb` active; log shows switch, add, switch back |
| US5 `use` atomic | `TestUse_ ./internal/e2e/` | add `"config user.email"` to `git.fail`; account switched back |
| US6 remotes | `TestFixRemote_ ./internal/e2e/` and `TestParseRemote ./internal/gitops/` | set `origin` to a GitLab URL; `fix-remote` exits 1, no `set-url` |
| US7 hosts | `TestImportAll_ ./internal/e2e/` | `run import-all --hostname ghe.example.com` → exit 2, no `gh` call |
| US8 workspace | `TestWorkspace_ ./internal/e2e/` and `TestWorkspacePattern ./internal/gitops/` | run `workspace` twice; one `--add` in the log |
| US9 strict flags | `TestFlags_ ./internal/e2e/` and `TestParseArgs ./internal/cli/` | `run use work --globl` → exit 2, no calls in the log |
| US10 docs/release | `TestReadme ./internal/cli/`; CI and release workflows on push/tag | `git tag v0.0.0-rc1` on a fork and observe the release job |
| US11 hermetic | `TestHarness_ ./internal/e2e/` | run the suite with `HOME` read-only |
| US12 `use` origin | `TestUse_Warns\|TestUse_FixRemote ./internal/e2e/` | steps `use work` and `use work --fix-remote` above |
| US13 `remove` | `TestRemove_ ./internal/e2e/` | `remove work`; `$tmp/home/.ssh` unchanged |
| US14 `doctor` | `TestDoctor_ ./internal/e2e/` | change `ssh.aliases.github-work.login` to `zamyb`; `ssh auth` line fails, exit 1 |
| US15 list/status | `TestList_\|TestStatus_ ./internal/e2e/` | set `origin` to `git@github-me:...`; `status` prints `mismatch:` lines |

## 5. Read-only smoke test on a real machine

These commands never mutate anything (FR-068, FR-078, FR-077), so they are safe
against real state once the feature is built:

```bash
go install ./cmd/ghs
ghs version
ghs list
ghs status
ghs doctor --offline
ghs doctor          # performs one ssh -T to github.com through your alias; no writes
```

Expected: `list` marks your active profile; `status` prints zero `mismatch:`
lines in a repository whose `origin` uses the right alias; `doctor` exits `0`
with `0 fail`.

## 6. CI and release

- Open a pull request: `.github/workflows/ci.yml` must run `gofmt -l`,
  `go vet ./...`, and `go test -race ./...` on `ubuntu-latest`, `macos-latest`,
  and `windows-latest` using `go-version-file: go.mod`.
- Push a tag `vX.Y.Z`: `.github/workflows/release.yml` must publish a GitHub
  Release whose notes are the matching `CHANGELOG.md` section and which has six
  archives (`linux`, `darwin`, `windows` × `amd64`, `arm64`).
- `ghs update` on a machine with the previous release installed must print
  `updated ghs vA → vB`.
