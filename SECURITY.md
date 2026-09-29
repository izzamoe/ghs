# Security policy

`ghs` switches GitHub CLI accounts, writes Git identities, generates and
uploads SSH keys, appends to `~/.ssh/config`, and rewrites repository
remotes. A bug in any of these can send commits, pushes, or keys to the wrong
account, so security reports are welcome.

## Supported versions

Only the latest release receives fixes. There are no long-term support
branches; a fix ships in a new release, and older releases are not patched.

| Version | Supported |
|---------|-----------|
| latest release (currently `v0.5.x`) | yes |
| anything older | no; update with `ghs update` or a new release archive |

## Reporting a vulnerability

Please report privately; do not open a public issue with details.

1. Use GitHub's private vulnerability reporting for this repository:
   [https://github.com/izzamoe/ghs/security/advisories/new](https://github.com/izzamoe/ghs/security/advisories/new)
   (also reachable from the repository's **Security** tab, **Report a
   vulnerability**). Only you and the maintainer can see the report.
2. If that form is unavailable, open a public issue whose title and body are
   only the words `security report, please contact me`, with no details, so
   the maintainer can open a private channel with you from there (for example
   a draft security advisory that you are added to).

The project has no security email address; please do not look for one.

Include what you can:

- the `ghs version` output, your OS and architecture, and `gh --version`,
  `git --version`, `ssh -V`;
- the exact command and the smallest steps that reproduce the problem;
- what happened and what you expected, for example which account, identity,
  remote, or file was affected.

Never include private keys, tokens, or the contents of your real config
files; describe them or redact them.

Handling is best effort by a single maintainer; there is no response-time
promise.

## Scope

In scope are defects in `ghs` itself, for example:

- switching to, or leaving active, a GitHub CLI account other than the one
  the command names, or failing to restore the previous account;
- writing a wrong Git identity, or writing it to the wrong scope or
  repository;
- SSH key generation or upload: overwriting a key, uploading to the wrong
  account, or uploading anything other than the `.pub` file (`ghs` never
  reads private key material);
- writes to `~/.ssh/config`, the `ghs` config file, identity files, or global
  Git `includeIf` entries that lose data, inject configuration, or change
  entries `ghs` does not own;
- config file parsing that lets crafted values reach SSH config, Git config,
  or remote URLs;
- rewriting `origin` to a host other than the profile's alias for
  `github.com`;
- passing untrusted input to `gh`, `git`, `ssh`, or `ssh-keygen` in a way that
  changes their meaning (`ghs` runs them with argument vectors, never through
  a shell).

## Out of scope

- Vulnerabilities in the GitHub CLI, Git, OpenSSH, Go, or GitHub itself;
  report those to their projects.
- Problems that require an attacker who can already write to your home
  directory, your config files, or your `PATH`.
- Running `ghs` with `sudo` or as root, which the documentation tells you not
  to do.
- Hosts other than `github.com`, which `ghs` does not support.

## No bug bounty

There is no bug bounty and no paid reward of any kind.
