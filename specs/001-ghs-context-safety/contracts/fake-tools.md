# Contract: Fake `gh`, `git`, `ssh`, and `ssh-keygen` for end-to-end tests

**Feature**: `001-ghs-context-safety` | **Date**: 2026-09-28 | **Spec**: [spec.md](../spec.md)

The fakes live in `internal/e2e` and run when the test binary is invoked under
a tool's name (see research.md R10). They implement only the invocations the
real `ghs` makes; any other invocation exits `64` with
`fake <tool>: unsupported arguments: <argv>` so that an unexpected call fails
the test loudly.

## Environment

| Variable | Set by | Meaning |
|----------|--------|---------|
| `GHS_FAKE_STATE` | sandbox | path of the JSON state file (read on every call, rewritten on simulated mutations) |
| `GHS_FAKE_LOG` | sandbox | path of the JSON-lines invocation log (append-only) |
| `HOME`, `USERPROFILE`, `XDG_CONFIG_HOME` | sandbox | isolated directories |
| `PATH` | sandbox | the sandbox `bin` directory only |
| `GORACE` | sandbox | `atexit_sleep_ms=0`, so the race-built fakes do not sleep one second on exit (affects only race-built binaries) |
| `SYSTEMROOT`, `TEMP`, `TMP` | sandbox (Windows only) | `TEMP`/`TMP` point inside the sandbox |

## Invocation log

Every call appends one line before doing anything else:

```json
{"tool":"gh","args":["auth","switch","--hostname","github.com","--user","zamyb-work"],"dir":"/tmp/sandbox/repo"}
```

`dir` is the working directory of the call. An `env` field lists the
*names* (never values) of the environment variables the fake saw, so the
isolation tests can prove the environment is minimal. Tests use helpers `calls()`,
`callsOf(tool)`, `assertSequence(...)`, and `assertNoMutations()`.

**Mutating operations** (for `assertNoMutations`, FR-068, FR-078, FR-063):
`gh auth switch`, `gh auth logout`, `gh ssh-key add`, `git config` without
`--get`/`--get-all`/`--show-origin`/`--list`, `git remote set-url`,
`git clone`, and any `ssh-keygen` call.

## State schema

```json
{
  "gh": {
    "accounts": [
      {"login": "zamyb-work", "active": true,  "state": "success"},
      {"login": "zamyb",      "active": false, "state": "success"},
      {"login": "broken",     "active": false, "state": "error"}
    ],
    "users": {
      "zamyb-work": {"id": 101, "login": "zamyb-work", "name": "Izzam", "email": "",
                     "emails": [{"email": "work@example.com", "primary": true, "verified": true, "visibility": "private"}]}
    },
    "ssh_keys": {"zamyb-work": ["ssh-ed25519 AAAAC3... zamyb-work"]},
    "fail": ["auth switch --user broken", "ssh-key add"]
  },
  "git": {
    "repo": true,
    "toplevel": "/tmp/sandbox/repo",
    "origin": "git@github.com:acme/app.git",
    "local":  {"user.name": "Izzam", "user.email": "work@example.com"},
    "global": {"user.email": "me@example.com",
               "includeIf.gitdir/i:~/Documents/work/.path": ["/tmp/sandbox/config/ghs/gitconfig-work"]},
    "fail": ["config user.email", "remote set-url"]
  },
  "ssh": {
    "aliases": {
      "github-work": {"login": "zamyb-work"},
      "github-me":   {"result": "denied"},
      "github-new":  {"result": "hostkey"},
      "github-slow": {"result": "timeout"}
    }
  },
  "keygen": {"fail": false}
}
```

Absent sections mean "tool present, empty state". A tool can be made *absent*
by the sandbox not creating its executable (`sandbox.without("gh")`).

`fail` entries are matched as substrings of the space-joined argument vector;
a match exits `1` with `fake <tool>: forced failure (<pattern>)` on stderr.

## Fake `gh`

| Invocation | Behavior |
|------------|----------|
| `auth status --hostname github.com --json hosts` | prints `{"hosts":{"github.com":[<accounts>]}}`; exits `1` with `You are not logged into any GitHub hosts` when `accounts` is empty; an account may carry `"host"` (default `github.com`), and when no account is for github.com it prints `{"hosts":{}}` |
| `auth status --hostname <other>` | prints `{"hosts":{}}` |
| `auth switch --hostname github.com --user <login>` | sets `active` on that account (must exist with state `success`, else exit `1` `could not switch`); persists state |
| `api user` | prints the `users[<active login>]` object (`id`, `login`, `name`, `email`); exit `1` if no active account |
| `api user/emails` | prints `users[<active>].emails` |
| `ssh-key add <file> --title <title>` | reads `<file>`; if its first two fields are already in `ssh_keys[<active>]`, prints `✓ Public key already exists on your account` and exits `0`; otherwise appends and prints `✓ Public key added to your account`; `<file>` unreadable → exit `1` |
| `auth logout ...` | always exits `1` `fake gh: logout must never be called` (guards FR-063) |

## Fake `git`

State is keyed by `toplevel`; `repo: false` makes `rev-parse` fail with exit
`128` `fatal: not a git repository`.

| Invocation | Behavior |
|------------|----------|
| `rev-parse --show-toplevel` | prints `toplevel` or fails |
| `remote get-url origin` | prints `origin` or exits `2` `error: No such remote 'origin'` when empty |
| `remote set-url origin <url>` | sets `origin`; persists |
| `config user.name <v>` / `config user.email <v>` | sets `local[...]`; requires `repo` |
| `config --global user.name <v>` / `user.email <v>` | sets `global[...]` |
| `-C <dir> config user.name/email <v>` | sets `local[...]` (dir recorded in log) |
| `config [--global] --show-origin --show-scope --get <key>` | resolves in order: local (`file:<toplevel>/.git/config`, scope `local`) unless `--global`; then an `includeIf.gitdir/i:<pattern>.path` entry whose expanded pattern is a prefix of the call's `dir` and whose file exists: reads `[user]` from that file, origin `file:<identity file>`, scope `global`; then `global[...]` (`file:<HOME>/.gitconfig`, scope `global`); prints `<scope>\t<origin>\t<value>` or exits `1` |
| `config --global --get-all <key>` | prints one value per line or exits `1` when absent |
| `config --global --add <key> <value>` | appends; persists |
| `config --global --unset --fixed-value <key> <value>` | removes exactly that value; exits `5` when absent (matches real Git) |
| `clone <url> <dir>` | creates `<dir>/.git/config` with `[remote "origin"] url = <url>`; sets `toplevel` to `<dir>` and `origin` to `<url>`; exits `128` when `<dir>` exists and is non-empty |

## Fake `ssh`

Only `-T -o BatchMode=yes -o ConnectTimeout=10 git@<alias>` is accepted.

| `aliases[<alias>]` | stderr | exit |
|--------------------|--------|------|
| `{"login": L}` | `Hi L! You've successfully authenticated, but GitHub does not provide shell access.` | `1` |
| `{"result": "denied"}` | `git@github.com: Permission denied (publickey).` | `255` |
| `{"result": "hostkey"}` | `Host key verification failed.` | `255` |
| `{"result": "timeout"}` | `ssh: connect to host github.com port 22: Connection timed out` | `255` |
| absent | `ssh: Could not resolve hostname <alias>: Name or service not known` | `255` |

## Fake `ssh-keygen`

Only `-t ed25519 -C <comment> -f <path> -N ""` is accepted. Creates `<path>`
(mode `0600`, content `FAKE PRIVATE KEY <comment>`) and `<path>.pub`
(`ssh-ed25519 AAAAFAKE <comment>`). Fails with exit `1` when `keygen.fail` is
true or `<path>` exists.

## Sandbox helper API (`internal/e2e/harness_test.go`)

```go
sb := newSandbox(t)                       // temp HOME, XDG_CONFIG_HOME, bin/, state, log
sb.state.GH.Accounts = ...                // mutate then sb.saveState()
sb.writeConfig(`[work]...`)               // <cfg>/ghs/config.conf
sb.writeSSHConfig("Host github-work\n...")
sb.touchKey("~/.ssh/id_ed25519_work")     // private + .pub
sb.without("gh")                          // remove the fake executable
res := sb.run("use", "work")              // res.Stdout, res.Stderr, res.Code
res := sb.runIn(sb.repoDir, "status")
sb.calls()                                // []call{Tool, Args, Dir}
sb.assertSequence(t, call{"gh", []string{"auth","switch",...}}, ...)
sb.assertNoMutations(t)
sb.snapshot("~/.ssh")                     // map[path]bytes for byte-identity checks (SC-010)
```
