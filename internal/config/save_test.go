package config

import (
	"errors"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func sampleConfig() Config {
	return Config{Profiles: []Profile{
		{Name: "work", GitHubUser: "zamyb-work", GitName: "Izzam", GitEmail: "work@example.com", SSHHostAlias: "github-work", SSHKey: "~/.ssh/id_ed25519_work", Workspace: "~/Documents/work",
			Extra: []KeyValue{{Key: "future_b", Value: "2"}, {Key: "future_a", Value: "1"}}},
		{Name: "me", GitHubUser: "zamyb", GitName: "Izzam", GitEmail: "me@example.com", SSHHostAlias: "github-me", SSHKey: "~/.ssh/id_ed25519_me"},
		{Name: "old", GitHubUser: "olduser", GitName: "Old", SSHHostAlias: "github-old", SSHKey: "~/.ssh/id_ed25519_old"},
	}}
}

func TestSaveIsAtomicRename(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.conf")
	original := "[keep]\ngh_user = \"keep\"\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	// A hard link to the original inode keeps the old bytes only if Save
	// replaced the directory entry instead of truncating the file in place.
	link := filepath.Join(dir, "link")
	if err := os.Link(path, link); err != nil {
		t.Skipf("hard links unsupported: %v", err)
	}
	if err := Save(path, sampleConfig()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	old, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(old) != original {
		t.Fatalf("original inode was modified in place: %q", old)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".config.conf-") {
			t.Fatalf("temporary file %s left behind", e.Name())
		}
	}
}

func TestSavePreservesMode(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	path := filepath.Join(t.TempDir(), "config.conf")
	if err := os.WriteFile(path, nil, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := Save(path, sampleConfig()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %v, want 0640", info.Mode().Perm())
	}
}

func TestSaveCreatesWithOwnerOnly(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission bits are not meaningful on Windows")
	}

	dir := filepath.Join(t.TempDir(), "nested", "ghs")
	path := filepath.Join(dir, "config.conf")
	if err := Save(path, sampleConfig()); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("file mode = %v, want 0600", info.Mode().Perm())
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %v, want 0700", dirInfo.Mode().Perm())
	}
}

func TestSavePreservesOrderAndExtraKeys(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.conf")
	cfg := sampleConfig()
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	want := `[work]
gh_user = "zamyb-work"
git_name = "Izzam"
git_email = "work@example.com"
ssh_host_alias = "github-work"
ssh_key = "~/.ssh/id_ed25519_work"
workspace = "~/Documents/work"
future_b = "2"
future_a = "1"

[me]
gh_user = "zamyb"
git_name = "Izzam"
git_email = "me@example.com"
ssh_host_alias = "github-me"
ssh_key = "~/.ssh/id_ed25519_me"

[old]
gh_user = "olduser"
git_name = "Old"
ssh_host_alias = "github-old"
ssh_key = "~/.ssh/id_ed25519_old"
`
	if string(data) != want {
		t.Fatalf("saved:\n%s\nwant:\n%s", data, want)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(loaded.Profiles, cfg.Profiles, profilesEqual) {
		t.Fatalf("round trip = %+v", loaded.Profiles)
	}
}

func TestSaveEmptyConfigWritesZeroBytes(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "config.conf")
	if err := Save(path, Config{}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil || info.Size() != 0 {
		t.Fatalf("stat = %v, %v; want zero-byte file", info, err)
	}
}

func profilesEqual(a, b Profile) bool {
	return a.Name == b.Name && a.GitHubUser == b.GitHubUser && a.GitName == b.GitName &&
		a.GitEmail == b.GitEmail && a.SSHHostAlias == b.SSHHostAlias && a.SSHKey == b.SSHKey &&
		a.Workspace == b.Workspace && slices.Equal(a.Extra, b.Extra)
}

// failingWriter accepts limit bytes and then fails.
type failingWriter struct {
	w     io.Writer
	limit int
}

func (f *failingWriter) Write(p []byte) (int, error) {
	if len(p) > f.limit {
		n, _ := f.w.Write(p[:f.limit])
		f.limit = 0
		return n, errors.New("injected write failure")
	}
	f.limit -= len(p)
	return f.w.Write(p)
}

func TestSaveLeavesOriginalOnWriteFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.conf")
	original := "[keep]\ngh_user = \"keep\"\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	err := saveWith(path, sampleConfig(), func(w io.Writer) io.Writer { return &failingWriter{w: w, limit: 17} })
	if err == nil {
		t.Fatal("saveWith() error = nil, want injected failure")
	}
	data, _ := os.ReadFile(path)
	if string(data) != original {
		t.Fatalf("original changed to %q", data)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("directory has %d entries, want only config.conf", len(entries))
	}
}

func TestSaveRandomInterruptsNeverCorrupt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.conf")
	if err := Save(path, sampleConfig()); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(path)
	next := sampleConfig()
	next.Profiles = append(next.Profiles, Profile{Name: "new", GitHubUser: "newuser", GitName: "New", SSHHostAlias: "github-new", SSHKey: "~/.ssh/id_new"})

	rng := rand.New(rand.NewPCG(1, 2))
	for i := range 100 {
		limit := rng.IntN(len(original) + 64)
		_ = saveWith(path, next, func(w io.Writer) io.Writer { return &failingWriter{w: w, limit: limit} })
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("run %d: %v", i, err)
		}
		cfg, err := Load(path)
		if err != nil {
			t.Fatalf("run %d (limit %d): config no longer parses: %v\n%s", i, limit, err, data)
		}
		if n := len(cfg.Profiles); n != 3 && n != 4 {
			t.Fatalf("run %d: %d profiles, want 3 (old) or 4 (new)", i, n)
		}
		if len(cfg.Profiles) == 4 {
			// A successful write happened; restore the old content for the next round.
			if err := os.WriteFile(path, original, 0o600); err != nil {
				t.Fatal(err)
			}
		} else if string(data) != string(original) {
			t.Fatalf("run %d: failed write changed bytes", i)
		}
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("directory has %d entries, want only config.conf", len(entries))
	}
}
