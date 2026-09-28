package config

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Save atomically replaces the config file with cfg: the complete content is
// written to a temporary file in the same directory, flushed, given the
// existing file's mode (or 0600 for a new file), and renamed over the target.
// A failure at any point leaves the previous file untouched.
func Save(path string, cfg Config) error {
	return saveWith(path, cfg, nil)
}

// saveWith is Save with an optional wrapper around the temporary file's
// writer, so tests can inject write failures.
func saveWith(path string, cfg Config, wrap func(io.Writer) io.Writer) error {
	var buf bytes.Buffer
	encode(&buf, cfg)
	if err := writeFileAtomic(path, buf.Bytes(), 0o600, wrap); err != nil {
		return fmt.Errorf("save config %s: %w", path, err)
	}
	return nil
}

func encode(w io.Writer, cfg Config) {
	bw := bufio.NewWriter(w)
	writePair := func(key, value string) {
		if value == "" {
			return
		}
		_, _ = bw.WriteString(key)
		_, _ = bw.WriteString(` = "`)
		_, _ = bw.WriteString(value)
		_, _ = bw.WriteString("\"\n")
	}
	for i, profile := range cfg.Profiles {
		if i > 0 {
			_ = bw.WriteByte('\n')
		}
		_ = bw.WriteByte('[')
		_, _ = bw.WriteString(profile.Name)
		_, _ = bw.WriteString("]\n")
		writePair("gh_user", profile.GitHubUser)
		writePair("git_name", profile.GitName)
		writePair("git_email", profile.GitEmail)
		writePair("ssh_host_alias", profile.SSHHostAlias)
		writePair("ssh_key", profile.SSHKey)
		writePair("workspace", profile.Workspace)
		for _, kv := range profile.Extra {
			writePair(kv.Key, kv.Value)
		}
	}
	_ = bw.Flush()
}

// writeFileAtomic replaces path with data via a sibling temporary file and a
// rename. An existing file keeps its permission bits; a new one gets
// newMode. The parent directory is created with mode 0700 when missing.
func writeFileAtomic(path string, data []byte, newMode fs.FileMode, wrap func(io.Writer) io.Writer) (err error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}
	mode := newMode
	if info, statErr := os.Stat(path); statErr == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(statErr, fs.ErrNotExist) {
		return statErr
	}

	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+"-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	var w io.Writer = tmp
	if wrap != nil {
		w = wrap(tmp)
	}
	if _, err = w.Write(data); err != nil {
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("flush temporary file: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err = os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("set mode: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
