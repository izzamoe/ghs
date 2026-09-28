package e2e

import (
	"errors"
	"io/fs"
	"os"
)

// keygen implements the "Fake ssh-keygen" table of contracts/fake-tools.md.
func (f *fakeCtx) keygen(args []string) int {
	if len(args) != 8 || args[0] != "-t" || args[1] != "ed25519" || args[2] != "-C" ||
		args[4] != "-f" || args[6] != "-N" || args[7] != "" {
		return f.unsupported(args)
	}
	comment, path := args[3], args[5]
	if f.st.Keygen.Fail {
		return f.failf(1, "fake ssh-keygen: forced failure")
	}
	if _, err := os.Stat(path); err == nil || !errors.Is(err, fs.ErrNotExist) {
		return f.failf(1, "%s already exists.", path)
	}
	if err := os.WriteFile(path, []byte("FAKE PRIVATE KEY "+comment+"\n"), 0o600); err != nil {
		return f.failf(1, "Saving key \"%s\" failed: %v", path, err)
	}
	if err := os.WriteFile(path+".pub", []byte("ssh-ed25519 AAAAFAKE "+comment+"\n"), 0o644); err != nil {
		return f.failf(1, "Saving key \"%s.pub\" failed: %v", path, err)
	}
	return 0
}
