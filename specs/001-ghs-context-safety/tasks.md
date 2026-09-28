---

description: "Task list for ghs Account-Context Safety"
---

# Tasks: ghs Account-Context Safety

**Input**: Design documents from `/specs/001-ghs-context-safety/`

**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli.md, contracts/fake-tools.md, quickstart.md

**Tests**: REQUIRED. The constitution (Principle II) and Story 11 mandate test-first,
hermetic end-to-end tests. Every implementation task names the test(s) that must be
written and observed failing before it. Test tasks precede implementation tasks in
each phase.

**Organization**: Phase 1 setup, Phase 2 foundational, then one phase per user story
in priority order (P1 before P2), then polish. Story numbers (US1–US15) are the
stable identifiers from spec.md; phase order is by priority and dependency.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: can run in parallel (different files, no dependency on an incomplete task)
- **[Story]**: user story label (US1…US15) for story-phase tasks only
- Every task names the exact file(s) it touches; behavior changes name the test that must fail first

## Path Conventions

Single Go module at the repository root. Production code in `cmd/ghs/` and
`internal/<pkg>/`; unit tests beside the code; end-to-end tests in
`internal/e2e/`; build/CI files at the root and under `.github/workflows/`.

---

## Phase 1: Setup (shared infrastructure and quality gates)

**Purpose**: local gates, CI, and the hermetic harness every story's tests need.

- [ ] T001 Create `Makefile` with targets `fmt` (`gofmt -l .` must print nothing, fail otherwise), `vet` (`go vet ./...`), `test` (`go test -race -count=1 ./...`), `e2e` (`go test -race -count=1 ./internal/e2e/...`), `check` (fmt, vet, test); document them in a comment header
- [ ] T002 [P] Create `.github/workflows/ci.yml`: triggers `pull_request` and `push` to `main`; matrix `ubuntu-latest`, `macos-latest`, `windows-latest`; `actions/setup-go` with `go-version-file: go.mod`; steps: `gofmt -l` (fail on output), `go vet ./...`, `go test -race -count=1 ./...`; `-buildvcs=false` not needed for tests
- [ ] T003 [P] Create `internal/e2e/main_test.go`: `TestMain` that (a) dispatches to a fake when `strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")` is `gh`, `git`, `ssh`, or `ssh-keygen`, and (b) otherwise builds the binary once with `go build -buildvcs=false -o <tmp>/ghs[.exe] github.com/izzamoe/ghs/cmd/ghs` into a `t.TempDir`-like directory removed on exit, storing the path in a package variable
- [ ] T004 [P] Create `internal/e2e/fake_state_test.go`: Go types mirroring the state schema in `contracts/fake-tools.md` (`fakeState{GH, Git, SSH, Keygen}`), `loadState()`/`saveState()` via `GHS_FAKE_STATE`, `appendLog(tool, args, dir)` writing one JSON line to `GHS_FAKE_LOG`, `failMatches(patterns, args) bool` (substring of space-joined args)
- [ ] T005 [P] Create `internal/e2e/fake_gh_test.go` implementing every row of the "Fake `gh`" table in `contracts/fake-tools.md` (`auth status --json hosts`, `auth switch`, `api user`, `api user/emails`, `ssh-key add` with already-present detection, `auth logout` always failing, unsupported → exit 64)
- [ ] T006 [P] Create `internal/e2e/fake_git_test.go` implementing every row of the "Fake `git`" table (`rev-parse --show-toplevel`, `remote get-url/set-url`, `config` local/global set, `-C <dir> config`, `config --show-origin --show-scope --get` with includeIf resolution reading `[user]` from the identity file, `--get-all`, `--add`, `--unset --fixed-value` exit 5 when absent, `clone`; persistence of mutations; unsupported → exit 64)
- [ ] T007 [P] Create `internal/e2e/fake_ssh_test.go` implementing the "Fake `ssh`" table (accepts only `-T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>`; outcomes `login`, `denied`, `hostkey`, `timeout`, absent; exact stderr strings and exit codes)
- [ ] T008 [P] Create `internal/e2e/fake_keygen_test.go` implementing the "Fake `ssh-keygen`" table (creates private `0600` and `.pub`; fails when `keygen.fail` or the path exists)
- [ ] T009 Create `internal/e2e/harness_test.go`: `newSandbox(t)` (temp `HOME`/`USERPROFILE`, `XDG_CONFIG_HOME`, `bin/` with copies or hard links of `os.Executable()` named `gh`, `git`, `ssh`, `ssh-keygen` plus `.exe` on Windows, empty state and log), `run`, `runIn`, `writeConfig`, `readConfig`, `writeSSHConfig`, `sshConfig`, `touchKey`, `without(tool)`, `calls`, `callsOf`, `assertSequence`, `assertNoMutations` (mutating list from `contracts/fake-tools.md`), `snapshot(dir)`; `run` passes an environment containing only `PATH=<bin>`, `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME`, `GHS_FAKE_STATE`, `GHS_FAKE_LOG`, and on Windows `SYSTEMROOT`/`TEMP`
- [ ] T010 Create `internal/e2e/fakes_selftest_test.go`: `TestFakeGH_AuthStatusAndSwitch`, `TestFakeGH_SSHKeyAddAlreadyPresent`, `TestFakeGit_ConfigShowOriginResolvesIncludeIf`, `TestFakeGit_UnsetFixedValueExit5`, `TestFakeSSH_Outcomes`, `TestFakeKeygen_CreatesPair`, `TestFakes_UnsupportedArgsExit64` — each invokes the fake through `sb.bin` directly and checks stdout/stderr/exit and persisted state
- [ ] T011 Create `internal/e2e/smoke_test.go` with `TestSmoke_VersionRunsInSandbox` (runs `ghs version`, exit 0, output starts with `ghs `) to prove T003 and T009 work on all three CI platforms

**Checkpoint**: `make check` passes; the sandbox and fakes are usable by every later phase.

---

## Phase 2: Foundational (blocking prerequisites)

**Purpose**: usage errors and exit codes, the strict parser, strict/atomic config, validation, and shared tool helpers that every story builds on.

- [ ] T012 Write `internal/cli/args_test.go`: `TestParseArgs` table covering unknown flag, missing value at end, missing value before another flag, duplicate flag, value on boolean flag (`--global yes`), surplus positional, too few positionals, flags before positionals, `--` terminator, `--help`/`-h` detection; each case asserts a `*UsageError` (or nil) and the parsed result — MUST FAIL until T014
- [ ] T013 Create `internal/cli/usage.go`: `type UsageError struct{ Command, Message, Usage string }` with `Error()`; `usageFor(cmd string) string` built from the command table; `App.PrintError` prints `ghs: <message>` then the usage block for usage errors
- [ ] T014 Create `internal/cli/args.go`: `type flagSpec{Name string; TakesValue bool}`, `type cmdSpec{Name string; MinPos, MaxPos int; Flags []flagSpec; Usage string}`, `parseArgs(spec cmdSpec, args []string) (pos []string, flags map[string]string, err error)` satisfying T012
- [ ] T015 Update `cmd/ghs/main.go`: `errors.As(err, &usageErr)` → `os.Exit(2)`; any other error → `os.Exit(1)`; print via `app.PrintError` (tests: `TestFlags_UnknownCommandExit2` and `TestFlags_UsageErrorNoSideEffects` in T056, written in Phase 12; this mapping is needed by every earlier e2e test that asserts exit codes, so implement it here)
- [ ] T016 Rewrite `internal/cli/app.go`: `var commands = []cmdSpec{...}` for every command in `contracts/cli.md` (including `doctor`, `remove`, `workspace`, `use --fix-remote`, `--workspace` on `add-profile`/`add-from-gh`); `Run` looks up the spec, handles `--help`, calls `parseArgs`, dispatches; `printHelp` renders the command list verbatim from `contracts/cli.md` plus `Config:`, `Only github.com is supported.`, and `Do not run ghs with sudo.` lines; `loadConfig` treats a missing file as empty and every other error as fatal
- [ ] T017 [P] Extend `internal/config/load_test.go`: `TestLoadRejectsDuplicateSections` (names both line numbers, case-insensitive), `TestLoadRejectsKeyOutsideSection`, `TestLoadRejectsUnterminatedQuote`, `TestLoadRejectsControlCharacters`, `TestLoadPreservesUnknownKeysInOrder`, `TestLoadMissingFileIsNotFound` (returns `os.ErrNotExist`), `TestLoadKeepsCRLF` — MUST FAIL until T019
- [ ] T018 [P] Create `internal/config/save_test.go`: `TestSaveIsAtomicRename` (temp file in same dir, no `O_TRUNC` on target), `TestSavePreservesMode` (`0640` stays `0640`), `TestSaveCreatesWithOwnerOnly` (`0600` file, `0700` dir), `TestSavePreservesOrderAndExtraKeys`, `TestSaveLeavesOriginalOnWriteFailure` (inject a writer that fails mid-way via an unexported `saveTo(w io.Writer)` seam), `TestSaveRandomInterruptsNeverCorrupt` (100 runs with a failing writer at random offsets; original always parses — SC-001) — MUST FAIL until T020
- [ ] T019 Update `internal/config/profile.go` (`Workspace string`, `Extra []KeyValue`, `FindProfile`, `FindProfileFold(name) (Profile, bool)`, `ByLogin`, `ByEmail`, `ByAlias`, `ByWorkspace`) and `internal/config/load.go` (line-numbered strict parser per data-model §2; missing file → `Config{}, fs.ErrNotExist`-wrapped error distinguishable with `errors.Is`) to satisfy T017
- [ ] T020 Rewrite `internal/config/save.go`: `Save(path, cfg)` → `os.CreateTemp(dir, ".config.conf-*")`, write known keys then `Extra`, `Sync`, `Close`, `Chmod` to existing mode or `0600`, `os.Rename`; remove temp on any failure; directory `MkdirAll 0700`; satisfy T018
- [ ] T021 [P] Create `internal/config/validate_test.go`: table tests `TestValidateName`, `TestValidateAlias`, `TestValidateLogin`, `TestValidateGitName`, `TestValidateEmail`, `TestValidateKeyPath`, `TestValidateWorkspace`, `TestCheckUnique` (name case-insensitive, alias case-insensitive, workspace equal and nested) with at least 30 rejected inputs in total (SC-003) — MUST FAIL until T022
- [ ] T022 Create `internal/config/validate.go` implementing FR-007 to FR-012, FR-056, FR-061 exactly as tabulated in `data-model.md` §1, plus `commandNames` reserved list sourced from `internal/cli` via a package-level slice passed in (`ValidateName(name string, reserved []string)`) to avoid an import cycle
- [ ] T023 [P] Create `internal/runner/runner_test.go`: `TestLookPathFindsExecutableInDir`, `TestCombinedOutputReturnsOutputAndExitCode` (uses `os.Executable()` re-invoked with an env-selected helper mode), `TestRunInSetsWorkingDirectory` — MUST FAIL until T024
- [ ] T024 Extend `internal/runner/runner.go`: `LookPath(name) (string, error)`, `CombinedOutput(name, args...) (out string, code int, err error)` (exit code extracted from `*exec.ExitError`), `RunIn(dir, name, args...)`, `OutputIn(dir, ...)`; keep `Run`, `Output`, `OutputBytes`
- [ ] T025 [P] Extend `internal/ghops/user_test.go`: `TestActiveAccount`, `TestIsAuthenticated` (state `success` only), `TestAddSSHKeyAlreadyPresentFromStderr` (non-zero exit with `already` → `AlreadyPresent=true, err=nil`) — MUST FAIL until T026
- [ ] T026 Extend `internal/ghops/user.go`: `ActiveAccount(hostname) (login string, ok bool, err error)`, `IsAuthenticated(hostname, login) (bool, error)`, `WithAccount(hostname, target string, fn func() error) (restored bool, err error)` (no switch when already active; switch back on success and failure; `errors.Join`), `AddSSHKey(pubPath, title string) (alreadyPresent bool, err error)`; `runner` field typed as an interface `commandRunner` so unit tests can stub it
- [ ] T027 [P] Create `internal/gitops/remote_test.go`: `TestParseRemote` table (scp `git@github.com:o/r.git`, `ssh://git@github.com/o/r`, `https://github.com/o/r/`, `http://github.com/o/r.git`, `git@github-work:o/r.git` → `SSHHost`, `ssh://git@github-me/o/r.git` → `SSHHost`, `git@gitlab.com:g/r.git` → `SSHHost` host `gitlab.com`, `https://gitlab.com/g/r` → `Other`, garbage → `Other`; path normalization: trailing `/` removed, exactly one `.git`), `TestRewriteGitHubURL` and `TestCloneURL` moved from `git_test.go` and extended — MUST FAIL until T028
- [ ] T028 Create `internal/gitops/remote.go` with `type Remote struct{Kind RemoteKind; Host, Path, Raw string}`, `ParseRemote(url) (Remote, error)`, `(Remote) AliasURL(alias) string` (`git@<alias>:<path>`), and move `RewriteGitHubURL`, `CloneURL`, `CloneDirectory` here rebuilt on `ParseRemote`; delete them from `internal/gitops/git.go`; keep `internal/gitops/git_bench_test.go` compiling
- [ ] T029 [P] Extend `internal/gitops/git_test.go`: `TestParseShowOriginLine` (`local\tfile:/r/.git/config\tIzzam` → scope, file, value) — MUST FAIL until T030
- [ ] T030 Extend `internal/gitops/git.go`: `InRepo(dir) (toplevel string, ok bool, err error)` via `rev-parse --show-toplevel`, `IdentityWithOrigin(dir string, global bool) (Identity{Name, Email, NameFile, EmailFile, Scope}, error)` via `config --show-origin --show-scope --get`, `SetIdentity(profile, global) (step string, err error)` reporting which key failed, `SetOriginURL`, `OriginURL`, `Clone` unchanged; `runner` typed as an interface for stubbing
- [ ] T031 Create `internal/cli/resolve.go` + `internal/cli/resolve_test.go` (`TestResolveByLogin`, `TestResolveByEmail`, `TestResolveByAlias`, `TestResolveOrigin` covering `profile`, `none`, `unaliased`, `notgithub`, `unknownalias`): `type resolution struct{Profile, Kind string}` and helpers used by `list`, `status`, `doctor`, `use`, `fix-remote`

**Checkpoint**: parser, exit codes, strict atomic config, validation, and tool helpers exist with unit tests; `go test ./...` green.

---

## Phase 3: User Story 11 — Hermetic test harness proven (Priority: P1)

**Goal**: the harness built in Phase 1 demonstrably isolates the developer's machine and runs on Windows.

**Independent Test**: run the suite with `HOME` read-only or empty; it passes.

### Tests for User Story 11

- [ ] T032 [P] [US11] Create `internal/e2e/isolation_test.go`: `TestHarness_EnvIsMinimal` (the environment passed to `ghs` contains only the allowed variables and `PATH` equals the sandbox `bin`), `TestHarness_SandboxHomeDiffersFromRealHome`, `TestHarness_LogRecordsArgsAndDir` (a `status` run in `sb.repoDir` logs `dir` equal to that directory), `TestHarness_FakesAreFoundOnWindows` (`runtime.GOOS == "windows"` → executables have `.exe` and are invoked), `TestHarness_NothingWrittenOutsideSandbox` (snapshot the sandbox parent temp root before/after a full `add-profile`, `init-ssh`, `workspace` flow and assert every changed path is under the sandbox)

### Implementation for User Story 11

- [ ] T033 [US11] Fix any harness gap found by T032 in `internal/e2e/harness_test.go` and `internal/e2e/main_test.go` (e.g., `USERPROFILE`, `SYSTEMROOT`, `TEMP` on Windows; hard-link fallback to copy)

**Checkpoint**: US11 scenarios 1–4 pass; scenario 5 is satisfied cumulatively by the story phases below.

---

## Phase 4: User Story 1 — Saved profiles are never lost (Priority: P1)

**Goal**: no profile-writing command can truncate, replace, or drop existing profiles.

**Independent Test**: seed three profiles, corrupt one line, run each writing command; file byte-identical, exit 1 naming the line.

### Tests for User Story 1

- [ ] T034 [P] [US1] Create `internal/e2e/config_test.go`: `TestConfig_InvalidLineStopsEveryWriter` (table over `add-profile`, `add-from-gh`, `import-all`, `set-email`, `workspace`, `remove`; exit 1, message contains `line 3`, file bytes unchanged, no `gh`/`git` mutation calls), `TestConfig_UnreadableFileNotReplaced` (chmod 000 on Unix; skipped on Windows with reason), `TestConfig_AppendPreservesOrderAndFields`, `TestConfig_CreatesDirAndFileOwnerOnly` (`0700`/`0600` on Unix), `TestConfig_UnknownKeysSurviveRoundTrip`

### Implementation for User Story 1

- [ ] T035 [US1] Rewrite `saveProfiles`/`saveProfile` in `internal/cli/add_profile.go` to use `a.loadConfig()` (fatal on any error other than not-found), merge into the loaded `Config` (preserving `Extra` and, per FR-060, an existing `Workspace`), and call `config.Save`; make T034 pass
- [ ] T036 [US1] Update `internal/cli/set_email.go` to load strictly, validate the email (`config.ValidateEmail` → `UsageError`), save atomically, and print `set email for profile "<p>"`; make the `set-email` rows of T034 pass

**Checkpoint**: US1 scenarios 1–5 pass.

---

## Phase 5: User Story 2 — Invalid names and values are rejected (Priority: P1)

**Goal**: every value from flags, the GitHub CLI, or derivation is validated before any tool call or write.

**Independent Test**: each malformed value → exit 2, no file change, empty invocation log.

### Tests for User Story 2

- [ ] T037 [P] [US2] Create `internal/e2e/validation_test.go`: `TestValidation_AddProfileRejectsMalformed` (table: name with `]`, space, `/`, control char, leading `-`, `help`; alias with space, `#`, `"`, `github.com`; login `-bad-`, 40 chars; git name with `"`; emails `a@b`, `@x.com`, `a b@x.com`, `a@x.com\n`; key path `relative/key`, empty; exit 2, no config file created, `len(sb.calls()) == 0`), `TestValidation_GHValuesRejected` (fake `gh api user` returns login `bad\nname` → exit 1 naming `login`; no save), `TestValidation_CaseInsensitiveDuplicateNameSuggests` (`Work` vs `work`), `TestValidation_AliasCollisionRejected`, `TestValidation_DerivedAliasCollision` (`add-from-gh` and `import-all` deriving an existing alias → exit 1, restore switch recorded for `import-all`), `TestValidation_SetEmailRejectsMalformed`

### Implementation for User Story 2

- [ ] T038 [US2] Wire validation into `internal/cli/add_profile.go`: `addProfile` validates every flag value and returns `UsageError`; `profileFromGH` validates GitHub CLI values and derived alias/key path (returning a plain error, exit 1); `validateImportedProfile`/`validateCompleteProfile` call `cfg.CheckUnique`; remove `unsafeProfilePathChars` in favor of `config.SafeSuffix(name)`; make T037 pass

**Checkpoint**: US2 scenarios 1–7 pass; SC-003 table count ≥ 30 across T021 and T037.

---

## Phase 6: User Story 3 — SSH config is safe with spaces in paths (Priority: P1)

**Goal**: quoted `IdentityFile` when needed, idempotent block, read-before-keygen.

**Independent Test**: sandbox home with a space; `init-ssh` writes one quoted block.

### Tests for User Story 3

- [ ] T039 [P] [US3] Extend `internal/sshops/ssh_test.go`: `TestEnsureConfigQuotesPathWithWhitespace`, `TestEnsureConfigUnquotedWithoutWhitespace`, `TestEnsureConfigLeavesFileUntouchedWhenHostPresent` (mtime and bytes equal; `Host a github-WORK b` matches case-insensitively), `TestEnsureConfigStartsBlockOnNewLine` (file without trailing newline), `TestEnsureConfigFailsWhenUnreadable` (Unix only), `TestEnsureKeyFailsWhenPublicKeyMissing`
- [ ] T040 [P] [US3] Create `internal/e2e/sshconfig_test.go`: `TestInitSSH_SpaceInHomeQuotesIdentityFile`, `TestInitSSH_SecondRunAddsNoBlock` (one `ssh-keygen` call total, one block), `TestInitSSH_UnreadableConfigNoKeygen` (Unix; zero `ssh-keygen` calls), `TestInitSSH_PrivateWithoutPublicFails` (no `ssh-keygen` call, private key bytes unchanged)

### Implementation for User Story 3

- [ ] T041 [US3] Update `internal/sshops/ssh.go`: `EnsureConfig` reads the file first (any error other than not-exist is fatal), checks `hasHostBlock`, builds the block with `quoteIfNeeded(filepath.ToSlash(keyPath))`, prepends `\n` only when the existing content is non-empty and lacks a trailing newline, appends; `EnsureKey` fails when the private key exists without `.pub`; make T039 pass
- [ ] T042 [US3] Reorder `internal/cli/init_ssh.go`: read SSH config (via `sshops.ReadConfig`) before `EnsureKey`; make T040 pass

**Checkpoint**: US3 scenarios 1–5 pass.

---

## Phase 7: User Story 4 — Upload goes to the right account and the previous account is restored (Priority: P1)

**Goal**: `init-ssh --upload` and `clone --upload-key` upload while the profile's account is active; `init-ssh` restores.

**Independent Test**: fake `gh` with `me` active; log shows switch → add → switch back.

### Tests for User Story 4

- [ ] T043 [P] [US4] Create `internal/e2e/upload_test.go`: `TestInitSSHUpload_SwitchesUploadsRestores` (exact sequence), `TestInitSSHUpload_FailureRestoresAndReportsBoth` (`ssh-key add` in `gh.fail`; stderr names both; exit 1; active account back to original), `TestInitSSHUpload_UnauthenticatedFailsBeforeKeygen`, `TestInitSSHUpload_NoSwitchWhenAlreadyActive`, `TestInitSSHUpload_AlreadyRegisteredIsSuccess` (stdout `already registered`), `TestCloneUploadKey_UploadsWhileSwitchedAndStaysSwitched`, `TestInitSSHUpload_UnreadablePubFailsBeforeSwitch`

### Implementation for User Story 4

- [ ] T044 [US4] Rewrite `internal/cli/init_ssh.go`: preflight (`config.ExpandPath`, SSH config readable, with `--upload`: `ghops.IsAuthenticated`, pub readable if key exists), then `EnsureKey`, `EnsureConfig`, then `ghops.WithAccount(profile.GitHubUser, upload)`; output lines from `contracts/cli.md`; make T043 pass
- [ ] T045 [US4] Update `internal/sshops/ssh.go` `UploadKey` to call `ghops.AddSSHKey` and return `alreadyPresent`; update `internal/cli/clone.go` to preflight authentication and upload after the switch (no restore); make the clone case of T043 pass

**Checkpoint**: US4 scenarios 1–6 pass.

---

## Phase 8: User Story 5 — `ghs use` leaves no partial state (Priority: P1)

**Goal**: preflight everything; roll back the account if the identity write fails; report account and scope.

**Independent Test**: fake `git` fails on `config user.email`; account switched back.

### Tests for User Story 5

- [ ] T046 [P] [US5] Create `internal/e2e/use_test.go`: `TestUse_NotInRepoFailsBeforeSwitch` (zero `gh auth switch` calls; message suggests `--global`), `TestUse_IdentityFailureRestoresAccount` (sequence switch → config name → config email(fail) → switch back; both errors on stderr), `TestUse_NoEmailSwitchesOnlyAccount` (stdout `git identity unchanged`), `TestUse_UnauthenticatedFailsBeforeAnything`, `TestUse_ReportsAccountAndLocalScope` (`git identity set: ... (local: <toplevel>)`), `TestUse_ReportsGlobalScope`, `TestUse_GlobalInsideRepoOnlyGlobal` (no non-`--global` config call), `TestUse_AlreadyActiveSkipsSwitch`

### Implementation for User Story 5

- [ ] T047 [US5] Rewrite `internal/cli/use.go` around a `usePlan` struct built in preflight (profile, target login, current login, toplevel or global, identity target) and an ordered `apply()` with reverse-order rollback via a `[]func() error` undo stack; output lines from `contracts/cli.md`; make T046 pass (the `--fix-remote` branch is added in Phase 9)

**Checkpoint**: US5 scenarios 1–5 pass.

---

## Phase 9: User Story 12 — `use` warns about an unaliased `origin`, or fixes it with `--fix-remote` (Priority: P1)

**Goal**: one stderr warning when `origin` bypasses the alias; `--fix-remote` rewrites first and rolls back on later failure.

**Independent Test**: fake `origin` `git@github.com:acme/app.git`; `use` warns and makes no `set-url` call; `use --fix-remote` records `set-url` and prints old → new.

### Tests for User Story 12

- [ ] T048 [P] [US12] Create `internal/e2e/use_remote_test.go`: `TestUse_WarnsWhenOriginIsGitHub` (exactly one `ghs: warning:` line naming the URL, `--fix-remote`, and `ghs fix-remote work`; zero `set-url`; exit 0 — SC-012), `TestUse_WarnsWhenOriginIsOtherProfileAlias` (names `me`), `TestUse_NoWarningWhenAliasCorrect`, `TestUse_NoWarningWithoutOrigin`, `TestUse_NoWarningOutsideRepoWithGlobal`, `TestUse_FixRemoteRewritesThenSwitchesThenSetsIdentity` (sequence `set-url` → `auth switch` → `config` ×2; stdout has `origin updated:`), `TestUse_FixRemoteRejectsNonGitHubBeforeMutation` (GitLab URL → exit 1 naming host; zero mutations), `TestUse_FixRemoteRejectsUnknownAlias`, `TestUse_FixRemoteMissingOriginFails`, `TestUse_FixRemoteOutsideRepoFailsEvenWithGlobal`, `TestUse_FixRemoteAlreadyCorrectNoSetURL` (stdout `origin already correct:`), `TestUse_FixRemoteRollbackRestoresOriginAndAccount` (`config user.email` fails → log ends with `auth switch` back and `set-url <old>`; exit 1; all results on stderr)

### Implementation for User Story 12

- [ ] T049 [US12] Extend `internal/cli/use.go`: preflight reads `origin` when in a repository (and requires a repository with `--fix-remote`), classifies via `resolve.go`/`gitops.ParseRemote`; with `--fix-remote` the first mutation is `SetOriginURL(new)` with undo `SetOriginURL(old)`; without it, after success, print the warning per FR-049; make T048 pass

**Checkpoint**: US12 scenarios 1–8 pass.

---

## Phase 10: User Story 6 — Non-GitHub remotes are left alone (Priority: P2)

**Goal**: only `github.com` URLs and known profile aliases are rewritten; `clone` rejects other hosts before mutating.

**Independent Test**: fake GitLab `origin` → `fix-remote` exits 1 naming the host, no `set-url`.

### Tests for User Story 6

- [ ] T050 [P] [US6] Create `internal/e2e/remote_test.go`: `TestFixRemote_RejectsGitLab`, `TestFixRemote_RewritesOtherProfileAlias`, `TestFixRemote_RejectsUnknownAlias`, `TestFixRemote_AlreadyCorrectNoSetURL`, `TestFixRemote_HTTPSAndSSHSchemeForms` (table incl. `.git` presence preserved), `TestFixRemote_TrailingSlashAndGitSuffixNormalized` (→ `git@github-work:owner/repo.git`), `TestFixRemote_NoOriginFails`, `TestFixRemote_WarnsWhenAliasMissingFromSSHConfig`, `TestClone_RejectsNonGitHubBeforeMutation` (zero `gh`/`ssh-keygen`/`git` calls)

### Implementation for User Story 6

- [ ] T051 [US6] Rewrite `internal/cli/fix_remote.go` on `resolve.go` (`unaliased` and `other profile alias` → rewrite; `profile` → already correct; `unknownalias`/`notgithub` → error naming the host); add the missing-SSH-block warning via `sshops.HasHostBlockInConfig`; make the `fix-remote` cases of T050 pass
- [ ] T052 [US6] Update `internal/cli/clone.go`: classify the input before any mutation (`owner/repo` shorthand or `Kind == GitHub` only); make `TestClone_RejectsNonGitHubBeforeMutation` pass

**Checkpoint**: US6 scenarios 1–7 pass.

---

## Phase 11: User Story 7 — Unsupported hosts are rejected (Priority: P2)

**Goal**: `--hostname` other than `github.com` is a usage error before any GitHub CLI call; docs say `github.com` only.

**Independent Test**: `import-all --hostname ghe.example.com` → exit 2, empty log.

### Tests for User Story 7

- [ ] T053 [P] [US7] Create `internal/e2e/hosts_test.go`: `TestImportAll_RejectsNonGitHubHostname` (exit 2, `len(sb.calls()) == 0`, no config file), `TestImportAll_AcceptsGitHubCaseInsensitive` (`--hostname GitHub.com` behaves as default)
- [ ] T054 [P] [US7] Add `TestHelpStatesGitHubOnly` to `internal/cli/docs_test.go` (help output contains `Only github.com is supported`)

### Implementation for User Story 7

- [ ] T055 [US7] Update `importAll` in `internal/cli/add_profile.go`: normalize and check `--hostname` before constructing `ghops.GH`; return `UsageError`; make T053 and T054 pass (help text already updated in T016)

**Checkpoint**: US7 scenarios 1–3 pass.

---

## Phase 12: User Story 9 — Mistyped flags and arguments are rejected (Priority: P2)

**Goal**: every command uses the strict parser; usage errors exit 2 with no side effects; `--help` per command.

**Independent Test**: unknown flag, missing value, duplicate, boolean-with-value, surplus positional on every command → exit 2, empty log, no files.

### Tests for User Story 9

- [ ] T056 [P] [US9] Create `internal/e2e/flags_test.go`: `TestFlags_UnknownFlagEveryCommand` (table over all 15 commands; stderr lists accepted flags), `TestFlags_MissingValueAtEndAndBeforeFlag`, `TestFlags_BooleanWithValue`, `TestFlags_DuplicateFlag`, `TestFlags_SurplusPositional` (table: `list`, `status`, `version`, `update`, `use`, `init-ssh`, `fix-remote`, `set-email`, `remove`, `doctor`, `workspace`), `TestFlags_BeforePositionalEquivalent` (`use --global work` produces the same log as `use work --global`), `TestFlags_DoubleDashEndsFlags`, `TestFlags_UsageErrorNoSideEffects` (empty log, no config/ssh files), `TestFlags_HelpPerCommandStdoutExit0`, `TestFlags_UnknownCommandExit2`, `TestFlags_RuntimeFailureExit1`

### Implementation for User Story 9

- [ ] T057 [US9] Convert every command handler to consume `(pos, flags)` from `parseArgs` and drop the ad-hoc `len(args)` checks: `internal/cli/add_profile.go`, `set_email.go`, `use.go`, `clone.go`, `init_ssh.go`, `fix_remote.go`, `list.go`, `status.go`, `update.go`; delete `parseFlags`, `hasFlag`, `hasFlagKey`, `cloneDirectoryArg`; update `internal/cli/flags_bench_test.go` to benchmark `parseArgs`; make T056 pass

**Checkpoint**: US9 scenarios 1–10 pass.

---

## Phase 13: User Story 8 — A directory is linked to a profile's Git identity (Priority: P2)

**Goal**: `workspace` links via identity file + `git config --global --add includeIf...`; idempotent; unlink exact; `--workspace` flag on add commands links after save.

**Independent Test**: run `workspace` twice → one identity file, one `--add`, zero on rerun; `--unlink` removes exactly that entry.

### Tests for User Story 8

- [ ] T058 [P] [US8] Create `internal/gitops/includeif_test.go`: `TestWorkspacePattern` (`~/Documents/work` → `~/Documents/work/`; `C:\Users\x\work` → `C:/Users/x/work/`; existing trailing `/` kept single; absolute Unix path unchanged plus `/`), `TestIncludeIfKey` (`includeIf.gitdir/i:~/Documents/work/.path`), `TestHasAddRemoveInclude` (stubbed runner: `--get-all` absent → add called once; present → not called; `--unset --fixed-value` exit 5 treated as success) — MUST FAIL until T061
- [ ] T059 [P] [US8] Create `internal/config/identity_test.go`: `TestIdentityFilePath` (`<cfg dir>/gitconfig-work`), `TestWriteIdentityFileAtomicAndMode0600`, `TestWriteIdentityFileExactContent`, `TestReadIdentityFile`, `TestIdentityFileUpToDate` — MUST FAIL until T060
- [ ] T060 [P] [US8] Create `internal/e2e/workspace_test.go`: `TestWorkspace_LinkWritesFileAddsIncludeSavesKey` (identity file content exact; log has `--get-all` then `--add`; config has `workspace`; three stdout lines — SC-011), `TestWorkspace_SecondRunIsIdempotent` (zero `--add`; stdout `already linked`), `TestWorkspace_RequiresEmail` (exit 1, zero mutations, names `set-email`), `TestWorkspace_RejectsInvalidPath` (table: relative, `~`, `~/`, `/`, newline, `"` → exit 2), `TestWorkspace_AddProfileFlagLinksAfterSave`, `TestWorkspace_AddFromGHFlagLinks`, `TestWorkspace_AddProfileLinkFailureReportsRerun` (`config --global --add` in `git.fail` → exit 1; config saved with `workspace`; stderr contains `rerun: ghs workspace work`), `TestWorkspace_UnlinkRemovesEntryFileAndKey` (exactly one `--unset --fixed-value`), `TestWorkspace_UnlinkWhenAlreadyAbsentSucceeds`, `TestWorkspace_LegacyKeyLoadsWithoutWarningAndListShowsIt`, `TestWorkspace_MissingDirectoryPrintsNote`, `TestWorkspace_SetEmailRefreshesIdentityFile`, `TestWorkspace_ImportAllOverwritePreservesWorkspaceAndRefreshesFile`, `TestWorkspace_ForeignIncludeEntryUntouched` (pre-seeded other value survives link and unlink), `TestWorkspace_RejectsSameAndNestedDirectory` (exit 1 naming the other profile), `TestWorkspace_RepairsPartialLink` (file-only and entry-only states → linked; output lists only what was added), `TestWorkspace_WindowsBackslashPathNormalized`

### Implementation for User Story 8

- [ ] T061 [US8] Create `internal/gitops/includeif.go`: `WorkspacePattern(path) string`, `IncludeIfKey(pattern) string`, `(Git) HasInclude(key, file) (bool, error)` (`--get-all`; exit 1 = absent), `(Git) AddInclude(key, file) error`, `(Git) RemoveInclude(key, file) error` (`--unset --fixed-value`; exit 5 = absent = success); make T058 pass
- [ ] T062 [US8] Create `internal/config/identity.go`: `IdentityFilePath(configPath, profileName) string`, `WriteIdentityFile(path, name, email) (changed bool, err error)` (atomic, `0600`, skips when content equal), `ReadIdentityFile(path) (name, email string, err error)`, `RemoveIdentityFile(path) error` (absent = success); make T059 pass
- [ ] T063 [US8] Create `internal/cli/workspace.go`: `workspaceLink(cfg, path, profile, wsPath)` used by `workspace`, `add-profile`, and `add-from-gh` (preflight: email, `ValidateWorkspace`, `CheckUnique` incl. nesting, `git` on `PATH`, `HasInclude` readable; mutations: identity file → `AddInclude` if missing → config save; output lines from `contracts/cli.md`; note when the expanded directory does not exist) and `workspaceUnlink(...)` (`RemoveInclude` → `RemoveIdentityFile` → config save); register `workspace` in `app.go`; make the `workspace` cases of T060 pass
- [ ] T064 [US8] Hook the link into `internal/cli/add_profile.go` (`--workspace` after a successful save; failure → exit 1 with the rerun message) and the identity-file refresh into `internal/cli/set_email.go` and `saveProfiles` (when the profile has a workspace and the identity file exists); make the remaining T060 cases pass

**Checkpoint**: US8 scenarios 1–11 pass; SC-011 holds.

---

## Phase 14: User Story 13 — `remove` deletes a profile without deleting anything unrecoverable (Priority: P2)

**Goal**: remove one section atomically; unlink workspace first; never touch keys, SSH config, or GitHub CLI logins; print what remains.

**Independent Test**: remove the middle of three profiles; other two byte-equivalent and ordered; `~/.ssh` and fake `gh` state byte-identical.

### Tests for User Story 13

- [ ] T065 [P] [US13] Create `internal/e2e/remove_test.go`: `TestRemove_MiddleProfilePreservesOthersAndExtraKeys`, `TestRemove_NotFoundExit1`, `TestRemove_NotFoundSuggestsCaseVariant` (`did you mean "Work"`), `TestRemove_KeepsSSHFilesAndConfigByteIdentical` (`sb.snapshot("~/.ssh")` before/after equal — SC-010), `TestRemove_NeverCallsLogoutOrSwitch` (only `gh auth status` in `callsOf("gh")`), `TestRemove_PrintsKeptLines` (key path, `Host github-old`, logged-in note with `gh auth logout ...`; `still active` variant), `TestRemove_UnlinksWorkspaceFirst` (log: `--unset` before config write; identity file gone; `unlinked workspace` line), `TestRemove_UnlinkFailureAbortsRemoval` (`--unset` in `git.fail` → exit 1; profile still present), `TestRemove_LastProfileLeavesEmptyFileWithMode`, `TestRemove_UsageErrors` (surplus arg, unknown flag → exit 2), `TestRemove_WorksWithoutGH` (`sb.without("gh")`; no account note; exit 0)

### Implementation for User Story 13

- [ ] T066 [US13] Create `internal/cli/remove.go`: strict load; exact lookup with `FindProfileFold` suggestion; optional read-only `gh auth status` for the note (skipped when `LookPath` fails or the call errors); `workspaceUnlink` when `Workspace != ""` (abort on error); `config.Save` with the section removed (zero-byte file when none remain); output per `contracts/cli.md`; register `remove` in `app.go`; make T065 pass

**Checkpoint**: US13 scenarios 1–8 pass; SC-010 holds.

---

## Phase 15: User Story 15 — `list` and `status` name the active profile and every mismatch (Priority: P2)

**Goal**: `list` marks the active profile and authentication state; `status` resolves three facts to profiles and prints `mismatch:` lines; both degrade without `gh`.

**Independent Test**: fake `gh` with `work` active and `old` absent → `*`, `active`/`yes`/`no`, trailing line; fake `origin` on the `me` alias → `mismatch:` lines.

### Tests for User Story 15

- [ ] T067 [P] [US15] Create `internal/e2e/list_test.go`: `TestList_MarksActiveAndAuthColumns` (exact table rows and trailing line), `TestList_ActiveMatchesNoProfile`, `TestList_WithoutGHShowsUnknown` (`?` column, `unknown (gh not found)`, exit 0), `TestList_GHFailureShowsUnknown` (`auth status` in `gh.fail`), `TestList_OtherHostsOnly` (`none for github.com`), `TestList_ShowsWorkspaceColumn`, `TestList_NoProfiles`, `TestList_NoMutations`
- [ ] T068 [P] [US15] Create `internal/e2e/status_test.go`: `TestStatus_AllResolveToSameProfileNoMismatch`, `TestStatus_OriginOtherProfileTwoMismatchLines` (order per contract), `TestStatus_IdentityNoProfile`, `TestStatus_AccountNoProfile`, `TestStatus_OriginGitHubDirectSuggestsFixRemote`, `TestStatus_OutsideRepoGlobalIdentityOriginUnavailable` (exit 0), `TestStatus_NotGitHubOrigin`, `TestStatus_ScopeWordAndWorkspaceWord` (identity from the identity file via fake includeIf → `global, workspace`), `TestStatus_WithoutGH`, `TestStatus_NoMutations`

### Implementation for User Story 15

- [ ] T069 [US15] Rewrite `internal/cli/list.go`: optional `ghops.AuthAccounts` (guarded by `LookPath`), columns and marker via `tabwriter`, `GH AUTH` values `active`/`yes`/`no`/`?`, trailing line variants; make T067 pass
- [ ] T070 [US15] Rewrite `internal/cli/status.go`: gather account (optional), `gitops.InRepo`, `IdentityWithOrigin` (global outside a repo), `OriginURL`; resolve each via `resolve.go`; print lines and `mismatch:` lines in the contract order; mark `workspace` when the email's origin file equals `config.IdentityFilePath` of the resolved profile; make T068 pass

**Checkpoint**: US15 scenarios 1–10 pass.

---

## Phase 16: User Story 14 — `doctor` explains the whole account context (Priority: P2)

**Goal**: six read-only checks with fixed outcomes, a summary, and exit 1 on any `fail`.

**Independent Test**: seed each of the eight mismatch classes; the corresponding line is `fail`/`warn`; the log has only read operations.

### Tests for User Story 14

- [ ] T071 [P] [US14] Create `internal/sshops/parse_test.go`: `TestParseHostBlock` (found/not found; `Host a github-work b`; case-insensitive keywords; `IdentityFile "/p/with space/id"` unquoted; `IdentityFile=~/.ssh/id` tilde expanded; stops at next `Host`/`Match`) — MUST FAIL until T074
- [ ] T072 [P] [US14] Create `internal/sshops/auth_test.go`: `TestClassifySSHOutput` table (`Hi zamyb-work! You've successfully authenticated...` → login; `Permission denied (publickey)` → denied; `Host key verification failed.` → hostkey; `Could not resolve hostname` → unresolvable; `Connection timed out` → timeout; other → unknown with first line) — MUST FAIL until T075
- [ ] T073 [P] [US14] Create `internal/e2e/doctor_test.go`: `TestDoctor_AllOKExit0` (six lines exact, `summary: 5 ok, 0 warn, 0 fail, 1 skip`), `TestDoctor_DefaultProfileFromActiveAccount`, `TestDoctor_NoMatchingProfileSkipsRest` (exit 1; five `skip ... no profile selected`), `TestDoctor_ExplicitProfileGHMismatchNamesBoth`, `TestDoctor_IdentityEmailFailNameWarn` (table; message has value, file, scope), `TestDoctor_OriginVariants` (table: github.com direct → warn; other profile alias → fail; unknown alias → fail; not GitHub → skip; no repo → skip; no origin → skip; correct → ok), `TestDoctor_SSHConfigVariants` (table: no file; no block; missing `HostName`; missing `User git`; missing `IdentitiesOnly yes`; `IdentityFile` missing; `.pub` missing; ok with quoted path), `TestDoctor_SSHAuthVariants` (table: login match ok; login mismatch fail; denied; hostkey with `run once: ssh -T git@github-work`; timeout; unresolvable), `TestDoctor_OfflineSkipsSSHAuth` (zero `ssh` calls), `TestDoctor_WorkspaceVariants` (table over the six link states in `data-model.md` §4), `TestDoctor_NoMutations` (`assertNoMutations`; sandbox snapshot unchanged — SC-009), `TestDoctor_MissingToolsFailButContinue` (`without("gh")`, `without("git")`, `without("ssh")` each), `TestDoctor_SummaryCountsAndExitCode`, `TestDoctor_UsageErrors`

### Implementation for User Story 14

- [ ] T074 [US14] Create `internal/sshops/parse.go`: `ReadConfig(home) (content string, path string, err error)`, `ParseHostBlock(content, alias) HostBlock`, `HasHostBlockInConfig(home, alias) (bool, error)`; make T071 pass
- [ ] T075 [US14] Create `internal/sshops/auth.go`: `type AuthResult struct{Kind AuthKind; Login, Message string}`, `ClassifySSHOutput(out string) AuthResult`, `(SSH) Verify(alias) (AuthResult, error)` running exactly `ssh -T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` via `runner.CombinedOutput`; make T072 pass
- [ ] T076 [US14] Create `internal/cli/doctor.go`: `doctorCheck{Name, Outcome, Message}`, profile selection per FR-069, the six checks per FR-070–075 using `ghops`, `gitops.IdentityWithOrigin`, `resolve.go`, `sshops.ParseHostBlock`, `sshops.Verify`, `gitops.HasInclude`, `config.ReadIdentityFile`; missing tools handled per FR-076; output and summary per `contracts/cli.md`; exit 1 via a non-usage error when any `fail`; register `doctor` in `app.go`; make T073 pass

**Checkpoint**: US14 scenarios 1–11 pass; SC-009 holds.

---

## Phase 17: User Story 10 — Documentation, versioning, release, CI, and license (Priority: P2)

**Goal**: README/help parity and Go-version parity enforced by tests; LICENSE, CHANGELOG, release pipeline.

**Independent Test**: `go test ./internal/cli -run TestReadme` passes; a tag on a fork produces a release with six archives.

### Tests for User Story 10

- [ ] T077 [P] [US10] Create `internal/cli/docs_test.go`: `TestReadmeGoVersionMatchesGoMod` (parses `go.mod`'s `go` directive and the README `Requires Go <ver>` line), `TestReadmeCommandListMatchesHelp` (the fenced command block in README equals the command list printed by `printHelp`, line for line), `TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` (README contains `exit code`, `Only github.com`, `sudo`), `TestHelpListsEveryRegisteredCommand` (each `cmdSpec.Name` appears in help) — MUST FAIL until T078–T079
- [ ] T078 [P] [US10] Create `internal/tools/changelog/main_test.go`: `TestExtractSectionForTag` (`v0.3.0` → the `## [0.3.0]` section body; `Unreleased` when asked; error when absent) — MUST FAIL until T081

### Implementation for User Story 10

- [ ] T079 [US10] Rewrite `README.md`: Go version from `go.mod`, the command block copied verbatim from `contracts/cli.md`, new sections for `doctor`, `remove`, `workspace`, `use --fix-remote`, `list`/`status` output, the exit-code contract (`0`/`1`/`2`), `Only github.com is supported`, the `sudo` warning, and the contributor section (`make check`, hermetic tests); make T077 pass
- [ ] T080 [P] [US10] Create `LICENSE` (MIT, copyright holder = repository owner `izzamoe`, year 2026) and `CHANGELOG.md` (Keep a Changelog header; `## [Unreleased]` with `### Added` — `remove`, `doctor`, `workspace`, `use --fix-remote`, `list`/`status` active profile and mismatch reporting, `origin` warning in `use`; `### Changed` — **Breaking:** strict flag parsing and exit code `2`; atomic config writes; validation; SSH quoting; upload account handling; `import-all --hostname` restriction; `### Fixed` — every defect from Stories 1–7)
- [ ] T081 [P] [US10] Create `internal/tools/changelog/main.go`: `go run ./internal/tools/changelog -tag v0.3.0 -file CHANGELOG.md` prints the section body to stdout; make T078 pass
- [ ] T082 [P] [US10] Create `.goreleaser.yaml` (builds `./cmd/ghs`, `goos` linux/darwin/windows, `goarch` amd64/arm64, `CGO_ENABLED=0`, `-buildvcs=false` not required for GoReleaser, archives with `LICENSE` and `README.md`, checksums) and `.github/workflows/release.yml` (on `push` tags `v*`; `actions/setup-go` with `go-version-file: go.mod`; `go run ./internal/tools/changelog -tag ${GITHUB_REF_NAME} > notes.md`; `goreleaser/goreleaser-action` with `args: release --clean --release-notes notes.md`)
- [ ] T083 [US10] Add the README/`go.mod` version check to CI by ensuring `go test ./...` in `.github/workflows/ci.yml` includes `internal/cli` (already true) and add a job step comment referencing `TestReadmeGoVersionMatchesGoMod`; verify `ghs version` prints the tag for a `go install` build by documenting the check in `README.md`

**Checkpoint**: US10 scenarios 1–7 pass; SC-005, SC-006, SC-007 hold.

---

## Phase 18: Polish & cross-cutting concerns

- [ ] T084 [P] Run `gofmt -w .` and `go vet ./...`; fix findings across `cmd/` and `internal/`
- [ ] T085 [P] Verify every existing test still passes and extend `internal/config/windows_test.go` and `internal/sshops/windows_test.go` for quoting and workspace pattern on Windows paths
- [ ] T086 Run the full `quickstart.md` section 3 script on Linux and (via CI) macOS and Windows; fix any output that differs from `contracts/cli.md`
- [ ] T087 Reconcile the traceability matrix below with the actual test names in the tree (`grep -rho 'func Test[A-Za-z0-9_]*' internal | sort`) and update this file
- [ ] T088 Update `CHANGELOG.md` `Unreleased` with anything discovered during implementation and confirm `README.md` still passes `TestReadmeCommandListMatchesHelp`
- [ ] T089 Final `make check` on a clean checkout with `HOME` pointed at an empty directory and the network disabled (e.g., `unshare -n` on Linux); all green

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: none; T001–T008 in parallel; T009 after T003–T008; T010–T011 after T009
- **Foundational (Phase 2)**: after Phase 1; blocks every story phase
- **US11 (Phase 3)**: after Phase 2; proves the harness before stories rely on it
- **US1, US2, US3, US4, US5 (Phases 4–8)**: after Phase 3; independent of one another except US4 uses `ghops.WithAccount` (T026) and US5 uses `gitops.SetIdentity` (T030)
- **US12 (Phase 9)**: after US5 (extends `use.go`) and T028/T031
- **US6 (Phase 10)**: after T028/T031; independent of US12 (both consume the classifier)
- **US7 (Phase 11)**: after Phase 2
- **US9 (Phase 12)**: after every command exists in its final shape for the flag table — run after Phases 4–11 but before Phases 13–16 add new commands, then re-run its table at the end of Phase 16 (the table iterates `commands`, so new commands are covered automatically)
- **US8 (Phase 13)**: after Phase 2 and T035/T036 (save hooks)
- **US13 (Phase 14)**: after US8 (uses `workspaceUnlink`)
- **US15 (Phase 15)**: after T031 and T030; `TestStatus_ScopeWordAndWorkspaceWord` needs US8's identity file
- **US14 (Phase 16)**: after US8, US15 (resolver), T074/T075
- **US10 (Phase 17)**: after all command phases (README must describe the final CLI)
- **Polish (Phase 18)**: last

### User Story Dependency Graph

```text
Setup → Foundational → US11
                         ├── US1 ─┐
                         ├── US2  │
                         ├── US3  │
                         ├── US4  │
                         ├── US5 ── US12
                         ├── US6  │
                         ├── US7  │
                         └── US9 (table re-run after new commands)
                                  │
                         US1 ──── US8 ── US13
                                   │
                                   ├── US15 ── US14
                                   │
                                   └──────────── US10 → Polish
```

### Parallel Opportunities

- Phase 1: T001–T008 concurrently (eight files)
- Phase 2: T017/T018/T021/T023/T025/T027/T029 (tests) concurrently, then their implementations
- Phases 4–8 and 10–11 can be assigned to different developers once Phase 3 is green
- Within each story, all `[P]` test tasks run concurrently before the implementation tasks

### Parallel Example: Phase 13 (US8)

```bash
# Tests first, together:
Task: "internal/gitops/includeif_test.go — TestWorkspacePattern, TestIncludeIfKey, TestHasAddRemoveInclude"
Task: "internal/config/identity_test.go — TestWriteIdentityFileAtomicAndMode0600, ..."
Task: "internal/e2e/workspace_test.go — TestWorkspace_*"
# Then implementation in dependency order: T061, T062 (parallel) → T063 → T064
```

---

## Implementation Strategy

### MVP First (P1 stories)

1. Phases 1–3: gates, harness, foundational packages, isolation proven
2. Phase 4 (US1) then Phase 5 (US2): the data-loss and corruption fixes
3. Phases 6–8 (US3, US4, US5): SSH, upload, `use` atomicity
4. Phase 9 (US12): `use` origin warning and `--fix-remote`
5. **STOP and VALIDATE**: `make check`; quickstart sections 1–3 for the P1 flows; tag a pre-release if desired

### Incremental Delivery

6. Phases 10–12 (US6, US7, US9): remote safety, host rejection, strict flags
7. Phase 13 (US8): workspace
8. Phases 14–16 (US13, US15, US14): `remove`, `list`/`status`, `doctor`
9. Phase 17 (US10): docs, license, changelog, release
10. Phase 18: polish, final hermetic run

---

## Traceability: functional requirement → test (SC-008)

| FR | Tests |
|----|-------|
| FR-001 | `TestConfig_InvalidLineStopsEveryWriter`, `TestConfig_UnreadableFileNotReplaced` |
| FR-002 | `TestSaveIsAtomicRename`, `TestSaveLeavesOriginalOnWriteFailure`, `TestSaveRandomInterruptsNeverCorrupt` |
| FR-003 | `TestSaveCreatesWithOwnerOnly`, `TestSavePreservesMode`, `TestConfig_CreatesDirAndFileOwnerOnly` |
| FR-004 | `TestLoadRejectsDuplicateSections`, `TestLoadRejectsKeyOutsideSection`, `TestLoadRejectsUnterminatedQuote`, `TestLoadRejectsControlCharacters` |
| FR-005 | `TestLoadPreservesUnknownKeysInOrder`, `TestConfig_UnknownKeysSurviveRoundTrip` |
| FR-006 | `TestSavePreservesOrderAndExtraKeys`, `TestConfig_AppendPreservesOrderAndFields` |
| FR-007 | `TestValidateName`, `TestValidation_AddProfileRejectsMalformed`, `TestValidation_CaseInsensitiveDuplicateNameSuggests` |
| FR-008 | `TestValidateAlias`, `TestValidation_AliasCollisionRejected` |
| FR-009 | `TestValidateLogin`, `TestValidation_GHValuesRejected` |
| FR-010 | `TestValidateGitName` |
| FR-011 | `TestValidateEmail`, `TestValidation_SetEmailRejectsMalformed` |
| FR-012 | `TestValidateKeyPath` |
| FR-013 | `TestValidation_AddProfileRejectsMalformed` (empty log), `TestValidation_GHValuesRejected` |
| FR-014 | `TestValidation_DerivedAliasCollision` |
| FR-015 | `TestEnsureConfigQuotesPathWithWhitespace`, `TestEnsureConfigUnquotedWithoutWhitespace`, `TestInitSSH_SpaceInHomeQuotesIdentityFile` |
| FR-016 | `TestEnsureConfigLeavesFileUntouchedWhenHostPresent`, `TestInitSSH_SecondRunAddsNoBlock` |
| FR-017 | `TestEnsureConfigStartsBlockOnNewLine`, `TestEnsureConfigLeavesFileUntouchedWhenHostPresent` |
| FR-018 | `TestEnsureConfigFailsWhenUnreadable`, `TestInitSSH_UnreadableConfigNoKeygen` |
| FR-019 | `TestEnsureKeyFailsWhenPublicKeyMissing`, `TestInitSSH_PrivateWithoutPublicFails` |
| FR-020 | `TestInitSSHUpload_UnauthenticatedFailsBeforeKeygen`, `TestUse_UnauthenticatedFailsBeforeAnything` |
| FR-021 | `TestInitSSHUpload_SwitchesUploadsRestores`, `TestInitSSHUpload_NoSwitchWhenAlreadyActive`, `TestInitSSHUpload_UnreadablePubFailsBeforeSwitch` |
| FR-022 | `TestCloneUploadKey_UploadsWhileSwitchedAndStaysSwitched` |
| FR-023 | `TestValidation_DerivedAliasCollision` (restore path), `TestImportAll_AcceptsGitHubCaseInsensitive` |
| FR-024 | `TestInitSSHUpload_FailureRestoresAndReportsBoth`, `TestUse_IdentityFailureRestoresAccount` |
| FR-025 | `TestAddSSHKeyAlreadyPresentFromStderr`, `TestInitSSHUpload_AlreadyRegisteredIsSuccess` |
| FR-026 | `TestUse_NotInRepoFailsBeforeSwitch`, `TestUse_UnauthenticatedFailsBeforeAnything` |
| FR-027 | `TestUse_IdentityFailureRestoresAccount` |
| FR-028 | `TestUse_ReportsAccountAndLocalScope`, `TestUse_ReportsGlobalScope`, `TestUse_NoEmailSwitchesOnlyAccount`, `TestUse_AlreadyActiveSkipsSwitch` |
| FR-029 | `TestParseRemote`, `TestFixRemote_HTTPSAndSSHSchemeForms`, `TestFixRemote_RewritesOtherProfileAlias`, `TestUse_FixRemoteRewritesThenSwitchesThenSetsIdentity` |
| FR-030 | `TestFixRemote_RejectsGitLab`, `TestFixRemote_RejectsUnknownAlias`, `TestFixRemote_NoOriginFails`, `TestClone_RejectsNonGitHubBeforeMutation` |
| FR-031 | `TestFixRemote_AlreadyCorrectNoSetURL` |
| FR-032 | `TestFixRemote_TrailingSlashAndGitSuffixNormalized`, `TestParseRemote` |
| FR-033 | `TestImportAll_RejectsNonGitHubHostname`, `TestImportAll_AcceptsGitHubCaseInsensitive` |
| FR-034 | `TestHelpStatesGitHubOnly`, `TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` |
| FR-035 | `TestWorkspace_AddProfileFlagLinksAfterSave`, `TestWorkspace_AddFromGHFlagLinks`, `TestWorkspace_LegacyKeyLoadsWithoutWarningAndListShowsIt` |
| FR-036 | `TestParseArgs`, `TestFlags_UnknownFlagEveryCommand`, `TestFlags_MissingValueAtEndAndBeforeFlag`, `TestFlags_DuplicateFlag`, `TestFlags_BooleanWithValue`, `TestFlags_SurplusPositional` |
| FR-037 | `TestFlags_BeforePositionalEquivalent`, `TestFlags_DoubleDashEndsFlags` |
| FR-038 | `TestFlags_UsageErrorNoSideEffects`, `TestFlags_UnknownCommandExit2`, `TestFlags_RuntimeFailureExit1` |
| FR-039 | `TestFlags_HelpPerCommandStdoutExit0` |
| FR-040 | `TestReadmeGoVersionMatchesGoMod` |
| FR-041 | `TestExtractSectionForTag` (changelog format); `LICENSE` presence checked in `TestReadmeStatesExitCodesGitHubOnlyAndNoSudo` |
| FR-042 | `.github/workflows/ci.yml` (T002); verified by CI run |
| FR-043 | `.github/workflows/release.yml`, `.goreleaser.yaml` (T082); `TestExtractSectionForTag` |
| FR-044 | `TestReadmeCommandListMatchesHelp`, `TestHelpListsEveryRegisteredCommand` |
| FR-045 | `TestSmoke_VersionRunsInSandbox`, `TestHarness_FakesAreFoundOnWindows` |
| FR-046 | `TestHarness_LogRecordsArgsAndDir`, `TestFakeGH_AuthStatusAndSwitch`, `TestFakeGit_ConfigShowOriginResolvesIncludeIf`, `TestFakeSSH_Outcomes` |
| FR-047 | `TestHarness_EnvIsMinimal`, `TestHarness_NothingWrittenOutsideSandbox`, `TestHarness_SandboxHomeDiffersFromRealHome` |
| FR-048 | every `internal/e2e` test listed in this matrix |
| FR-049 | `TestUse_WarnsWhenOriginIsGitHub`, `TestUse_WarnsWhenOriginIsOtherProfileAlias`, `TestUse_NoWarningWhenAliasCorrect`, `TestUse_NoWarningWithoutOrigin`, `TestUse_NoWarningOutsideRepoWithGlobal` |
| FR-050 | `TestUse_FixRemoteRejectsNonGitHubBeforeMutation`, `TestUse_FixRemoteRejectsUnknownAlias`, `TestUse_FixRemoteMissingOriginFails`, `TestUse_FixRemoteOutsideRepoFailsEvenWithGlobal` |
| FR-051 | `TestUse_FixRemoteRewritesThenSwitchesThenSetsIdentity`, `TestUse_FixRemoteRollbackRestoresOriginAndAccount` |
| FR-052 | `TestUse_FixRemoteAlreadyCorrectNoSetURL` |
| FR-053 | `TestList_ShowsWorkspaceColumn`, `TestStatus_ScopeWordAndWorkspaceWord`, `TestDoctor_WorkspaceVariants`, `TestRemove_UnlinksWorkspaceFirst` |
| FR-054 | `TestWorkspace_LinkWritesFileAddsIncludeSavesKey`, `TestWorkspacePattern`, `TestIncludeIfKey`, `TestIdentityFilePath`, `TestWriteIdentityFileAtomicAndMode0600`, `TestWriteIdentityFileExactContent` |
| FR-055 | `TestHasAddRemoveInclude` (only `git config --global` invocations), `TestWorkspace_LinkWritesFileAddsIncludeSavesKey` (no `.gitconfig` in sandbox home) |
| FR-056 | `TestValidateWorkspace`, `TestWorkspace_RejectsInvalidPath`, `TestWorkspace_WindowsBackslashPathNormalized` |
| FR-057 | `TestWorkspace_SecondRunIsIdempotent`, `TestWorkspace_RepairsPartialLink` |
| FR-058 | `TestWorkspace_UnlinkRemovesEntryFileAndKey`, `TestWorkspace_UnlinkWhenAlreadyAbsentSucceeds`, `TestWorkspace_ForeignIncludeEntryUntouched`, `TestFakeGit_UnsetFixedValueExit5` |
| FR-059 | `TestWorkspace_RequiresEmail` |
| FR-060 | `TestWorkspace_SetEmailRefreshesIdentityFile`, `TestWorkspace_ImportAllOverwritePreservesWorkspaceAndRefreshesFile` |
| FR-061 | `TestCheckUnique`, `TestWorkspace_RejectsSameAndNestedDirectory` |
| FR-062 | `TestRemove_MiddleProfilePreservesOthersAndExtraKeys`, `TestRemove_NotFoundExit1`, `TestRemove_NotFoundSuggestsCaseVariant` |
| FR-063 | `TestRemove_KeepsSSHFilesAndConfigByteIdentical`, `TestRemove_NeverCallsLogoutOrSwitch` |
| FR-064 | `TestRemove_UnlinksWorkspaceFirst`, `TestRemove_UnlinkFailureAbortsRemoval` |
| FR-065 | `TestRemove_PrintsKeptLines`, `TestRemove_WorksWithoutGH` |
| FR-066 | `TestRemove_LastProfileLeavesEmptyFileWithMode` |
| FR-067 | `TestDoctor_AllOKExit0`, `TestDoctor_SummaryCountsAndExitCode` |
| FR-068 | `TestDoctor_NoMutations` |
| FR-069 | `TestDoctor_DefaultProfileFromActiveAccount`, `TestDoctor_NoMatchingProfileSkipsRest` |
| FR-070 | `TestDoctor_ExplicitProfileGHMismatchNamesBoth`, `TestDoctor_MissingToolsFailButContinue` |
| FR-071 | `TestDoctor_IdentityEmailFailNameWarn`, `TestParseShowOriginLine` |
| FR-072 | `TestDoctor_OriginVariants` |
| FR-073 | `TestDoctor_SSHConfigVariants`, `TestParseHostBlock` |
| FR-074 | `TestDoctor_SSHAuthVariants`, `TestDoctor_OfflineSkipsSSHAuth`, `TestClassifySSHOutput`, `TestFakeSSH_Outcomes` (exact argv accepted) |
| FR-075 | `TestDoctor_WorkspaceVariants` |
| FR-076 | `TestDoctor_MissingToolsFailButContinue` |
| FR-077 | `TestList_MarksActiveAndAuthColumns`, `TestList_ActiveMatchesNoProfile`, `TestList_WithoutGHShowsUnknown`, `TestList_GHFailureShowsUnknown`, `TestList_OtherHostsOnly`, `TestList_NoMutations` |
| FR-078 | `TestStatus_AllResolveToSameProfileNoMismatch`, `TestStatus_OriginOtherProfileTwoMismatchLines`, `TestStatus_IdentityNoProfile`, `TestStatus_AccountNoProfile`, `TestStatus_OriginGitHubDirectSuggestsFixRemote`, `TestStatus_OutsideRepoGlobalIdentityOriginUnavailable`, `TestStatus_NoMutations` |
| FR-079 | `TestList_WithoutGHShowsUnknown`, `TestStatus_WithoutGH`, `TestRemove_WorksWithoutGH` |

## Notes

- Every implementation task above names the failing test it must turn green; write the test, run it, watch it fail, then implement (constitution Principle II).
- `[P]` tasks touch different files and have no dependency on an incomplete task.
- Commit after each task or logical group; never commit with `make check` failing.
- Stop at any checkpoint to validate the story independently with the quickstart.
