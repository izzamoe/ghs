# Getting help with ghs

## Help yourself first

Most problems can be diagnosed locally, without any network access:

1. Run `ghs doctor --offline` (or `ghs doctor <profile> --offline`). It runs
   the read-only checks and names the command that fixes each finding. Drop
   `--offline` to also test SSH authentication with GitHub.
2. Run `ghs status` to see whether the active GitHub CLI account, the Git
   identity, and `origin` resolve to the same profile.
3. Read the troubleshooting section of the README,
   [README.md#troubleshooting](README.md#troubleshooting): it lists common
   symptoms, their causes, the `ghs` fix, and how to undo every change `ghs`
   makes by hand with `gh`, `git`, and a text editor.
4. Run `ghs help <command>` for the exact flags of a command.
5. Before opening an issue, search the existing ones, open and closed:
   [https://github.com/izzamoe/ghs/issues?q=is%3Aissue](https://github.com/izzamoe/ghs/issues?q=is%3Aissue).

## Ask a question or report a bug

If that does not help, open an issue with the issue chooser at
[https://github.com/izzamoe/ghs/issues/new/choose](https://github.com/izzamoe/ghs/issues/new/choose):
use **Bug report** for something that does not work as documented and
**Feature request** for an idea or a question about intended behavior.

Include:

- the output of `ghs version`, your OS and architecture, `gh --version`, and
  `git --version`;
- the command you ran, what you expected, and what happened;
- the output of `ghs doctor --offline`.

Redact logins, emails, and paths if you prefer. Never paste private keys,
tokens, or full config files.

Security problems are not support questions: follow [SECURITY.md](SECURITY.md)
instead.

## What to expect

Support is best effort by one maintainer. There is no response-time promise,
no paid support, and no chat or email channel. Issues that include the information
above are much easier to act on.
