package gitops

import (
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestWorkspacePattern(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"~/Documents/work":   "~/Documents/work/",
		"~/Documents/work/":  "~/Documents/work/",
		"~/Documents/work//": "~/Documents/work/",
		`C:\Users\x\work`:    "C:/Users/x/work/",
		"/srv/work":          "/srv/work/",
	} {
		if got := WorkspacePattern(in); got != want {
			t.Errorf("WorkspacePattern(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIncludeIfKey(t *testing.T) {
	t.Parallel()
	if got := IncludeIfKey("~/Documents/work/"); got != "includeIf.gitdir/i:~/Documents/work/.path" {
		t.Fatalf("IncludeIfKey() = %q", got)
	}
}

// exitError carries an exit code like *exec.ExitError.
type exitError int

func (e exitError) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exitError) ExitCode() int { return int(e) }

// configStub models `git config --global` multi-valued keys.
type configStub struct {
	values  map[string][]string
	calls   [][]string
	failAll error
}

func (s *configStub) record(name string, args []string) error {
	s.calls = append(s.calls, append([]string{name}, args...))
	return s.failAll
}

func (s *configStub) Run(name string, args ...string) error {
	if err := s.record(name, args); err != nil {
		return err
	}
	switch {
	case len(args) == 5 && args[2] == "--add":
		s.values[args[3]] = append(s.values[args[3]], args[4])
		return nil
	case len(args) == 6 && args[2] == "--unset" && args[3] == "--fixed-value":
		i := slices.Index(s.values[args[4]], args[5])
		if i < 0 {
			return fmt.Errorf("run git: %w", exitError(5))
		}
		s.values[args[4]] = slices.Delete(s.values[args[4]], i, i+1)
		return nil
	}
	return fmt.Errorf("unexpected %v", args)
}

func (s *configStub) Output(name string, args ...string) (string, error) {
	if err := s.record(name, args); err != nil {
		return "", err
	}
	if len(args) == 4 && args[2] == "--get-all" {
		v := s.values[args[3]]
		if len(v) == 0 {
			return "", fmt.Errorf("run git: %w", exitError(1))
		}
		return strings.Join(v, "\n"), nil
	}
	return "", fmt.Errorf("unexpected %v", args)
}

func TestHasAddRemoveInclude(t *testing.T) {
	t.Parallel()
	key := IncludeIfKey(WorkspacePattern("~/Documents/work"))
	stub := &configStub{values: map[string][]string{key: {"/foreign/file"}}}
	g := New(stub)

	if has, err := g.HasInclude(key, "/cfg/gitconfig-work"); err != nil || has {
		t.Fatalf("HasInclude(absent) = %v, %v", has, err)
	}
	if err := g.AddInclude(key, "/cfg/gitconfig-work"); err != nil {
		t.Fatal(err)
	}
	if has, err := g.HasInclude(key, "/cfg/gitconfig-work"); err != nil || !has {
		t.Fatalf("HasInclude(present) = %v, %v", has, err)
	}
	if removed, err := g.RemoveInclude(key, "/cfg/gitconfig-work"); err != nil || !removed {
		t.Fatalf("RemoveInclude() = %v, %v", removed, err)
	}
	// Absent entry (exit 5) is success, and the foreign value is untouched.
	if removed, err := g.RemoveInclude(key, "/cfg/gitconfig-work"); err != nil || removed {
		t.Fatalf("RemoveInclude(absent) = %v, %v", removed, err)
	}
	if got := stub.values[key]; !slices.Equal(got, []string{"/foreign/file"}) {
		t.Fatalf("remaining values = %v", got)
	}
	for _, c := range stub.calls {
		if c[0] != "git" || c[1] != "config" || c[2] != "--global" {
			t.Fatalf("call %v is not git config --global", c)
		}
	}
	// An absent key is "not found", but any other failure is an error.
	empty := New(&configStub{values: map[string][]string{}})
	if has, err := empty.HasInclude(key, "/x"); err != nil || has {
		t.Fatalf("HasInclude(no key) = %v, %v", has, err)
	}
	broken := New(&configStub{values: map[string][]string{}, failAll: fmt.Errorf("run git: %w", exitError(3))})
	if _, err := broken.HasInclude(key, "/x"); err == nil {
		t.Fatal("HasInclude() hid a read failure")
	}
}
