// Package e2e holds the hermetic end-to-end suite for ghs.
//
// The test binary doubles as every fake tool: when it is executed under the
// name gh, git, ssh, or ssh-keygen it behaves as that fake (see
// specs/001-ghs-context-safety/contracts/fake-tools.md) instead of running
// tests. Otherwise TestMain builds the real ghs binary once and runs each test
// against it inside its own sandbox (temporary HOME, XDG_CONFIG_HOME, and a
// bin directory holding only the fakes), so nothing outside the sandbox is
// read or written and no network is used.
package e2e
