package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestIdentityFilePath(t *testing.T) {
	t.Parallel()
	cfg := filepath.Join("cfg", "ghs", "config.conf")
	if got, want := IdentityFilePath(cfg, "work"), filepath.Join("cfg", "ghs", "gitconfig-work"); got != want {
		t.Fatalf("IdentityFilePath() = %q, want %q", got, want)
	}
}

func TestWriteIdentityFileAtomicAndMode0600(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "gitconfig-work")

	changed, err := WriteIdentityFile(path, "Izzam", "work@example.com")
	if err != nil || !changed {
		t.Fatalf("WriteIdentityFile(new) = %v, %v", changed, err)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
		}
	}

	old := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if changed, err := WriteIdentityFile(path, "Izzam", "work@example.com"); err != nil || changed {
		t.Fatalf("WriteIdentityFile(same) = %v, %v; want untouched", changed, err)
	}
	if info, _ := os.Stat(path); !info.ModTime().Equal(old) {
		t.Fatal("up-to-date identity file was rewritten")
	}

	link := filepath.Join(dir, "link")
	if err := os.Link(path, link); err == nil {
		if changed, err := WriteIdentityFile(path, "Izzam", "new@example.com"); err != nil || !changed {
			t.Fatalf("WriteIdentityFile(changed) = %v, %v", changed, err)
		}
		if data, _ := os.ReadFile(link); string(data) != "[user]\n\tname = Izzam\n\temail = work@example.com\n" {
			t.Fatalf("identity file rewritten in place: %q", data)
		}
	}
}

func TestWriteIdentityFileExactContent(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ name, want string }{
		{"Izzam", "[user]\n\tname = Izzam\n\temail = work@example.com\n"},
		{"Zoë #1", "[user]\n\tname = \"Zoë #1\"\n\temail = work@example.com\n"},
		{`back\slash`, "[user]\n\tname = \"back\\\\slash\"\n\temail = work@example.com\n"},
	} {
		path := filepath.Join(t.TempDir(), "gitconfig-work")
		if _, err := WriteIdentityFile(path, tc.name, "work@example.com"); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if string(data) != tc.want {
			t.Errorf("content for %q = %q, want %q", tc.name, data, tc.want)
		}
	}
}

func TestReadIdentityFile(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"Izzam", "Zoë #1", `back\slash`} {
		path := filepath.Join(t.TempDir(), "gitconfig-x")
		if _, err := WriteIdentityFile(path, name, "a@example.com"); err != nil {
			t.Fatal(err)
		}
		gotName, gotEmail, err := ReadIdentityFile(path)
		if err != nil || gotName != name || gotEmail != "a@example.com" {
			t.Fatalf("ReadIdentityFile() = %q, %q, %v; want %q", gotName, gotEmail, err, name)
		}
	}
	if _, _, err := ReadIdentityFile(filepath.Join(t.TempDir(), "absent")); !os.IsNotExist(err) {
		t.Fatalf("ReadIdentityFile(absent) error = %v, want not-exist", err)
	}
}

func TestIdentityFileUpToDate(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "gitconfig-work")
	if ok, err := IdentityFileUpToDate(path, "Izzam", "work@example.com"); err != nil || ok {
		t.Fatalf("absent = %v, %v", ok, err)
	}
	if _, err := WriteIdentityFile(path, "Izzam", "work@example.com"); err != nil {
		t.Fatal(err)
	}
	if ok, err := IdentityFileUpToDate(path, "Izzam", "work@example.com"); err != nil || !ok {
		t.Fatalf("match = %v, %v", ok, err)
	}
	if ok, err := IdentityFileUpToDate(path, "Izzam", "other@example.com"); err != nil || ok {
		t.Fatalf("differs = %v, %v", ok, err)
	}
	if removed, err := RemoveIdentityFile(path); err != nil || !removed {
		t.Fatalf("RemoveIdentityFile() = %v, %v", removed, err)
	}
	if removed, err := RemoveIdentityFile(path); err != nil || removed {
		t.Fatalf("RemoveIdentityFile(absent) = %v, %v", removed, err)
	}
}
