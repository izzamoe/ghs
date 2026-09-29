## Summary

<!-- What changes, and why. Link the issue or the specs/<NNN-name>/ feature. -->

## Checklist

- [ ] `make check` passed locally (or `gofmt -l .`, `go vet ./...`, `go test -race -count=1 ./...` without `make`).
- [ ] For a behavior change or bug fix, a failing test was written first and now passes.
- [ ] `CHANGELOG.md` has an `## [Unreleased]` entry for every user-visible change (none needed for internal-only changes).
- [ ] For CLI changes, `README.md` and `--help` output were updated together (the Command reference is regenerated from `ghs <command> --help`).
- [ ] no new runtime dependency (or a justification is recorded in the feature's `research.md`).
- [ ] Constitution Principles I–III and VI were considered for changes that mutate user state, parse input, or change documentation.
- [ ] No private keys, tokens, real logins, or real emails appear in code, tests, or docs.
