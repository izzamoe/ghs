package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// fakeState mirrors the state schema in contracts/fake-tools.md. The fakes
// read it on every invocation and persist the mutations they simulate.
type fakeState struct {
	GH     ghState     `json:"gh"`
	Git    gitState    `json:"git"`
	SSH    sshState    `json:"ssh"`
	Keygen keygenState `json:"keygen"`
}

type ghAccount struct {
	Login  string `json:"login"`
	Active bool   `json:"active"`
	State  string `json:"state"`
	// Host defaults to github.com; any other value models an account the
	// GitHub CLI holds for a different host.
	Host string `json:"host,omitempty"`
}

type ghEmail struct {
	Email      string `json:"email"`
	Primary    bool   `json:"primary"`
	Verified   bool   `json:"verified"`
	Visibility string `json:"visibility,omitempty"`
}

type ghUser struct {
	ID     int64     `json:"id"`
	Login  string    `json:"login"`
	Name   string    `json:"name"`
	Email  string    `json:"email"`
	Emails []ghEmail `json:"emails,omitempty"`
}

type ghState struct {
	Accounts []ghAccount         `json:"accounts,omitempty"`
	Users    map[string]ghUser   `json:"users,omitempty"`
	SSHKeys  map[string][]string `json:"ssh_keys,omitempty"`
	Fail     []string            `json:"fail,omitempty"`
}

// gitValues is a multi-valued Git config entry. JSON accepts a single string
// or an array of strings.
type gitValues []string

func (v *gitValues) UnmarshalJSON(data []byte) error {
	var one string
	if err := json.Unmarshal(data, &one); err == nil {
		*v = gitValues{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(data, &many); err != nil {
		return err
	}
	*v = many
	return nil
}

type gitState struct {
	Repo     bool                 `json:"repo"`
	Toplevel string               `json:"toplevel,omitempty"`
	Origin   string               `json:"origin,omitempty"`
	Local    map[string]gitValues `json:"local,omitempty"`
	Global   map[string]gitValues `json:"global,omitempty"`
	Fail     []string             `json:"fail,omitempty"`
}

type sshAlias struct {
	Login  string `json:"login,omitempty"`
	Result string `json:"result,omitempty"`
}

type sshState struct {
	Aliases map[string]sshAlias `json:"aliases,omitempty"`
}

type keygenState struct {
	Fail bool `json:"fail,omitempty"`
}

func readStateFile(path string) (fakeState, error) {
	var st fakeState
	data, err := os.ReadFile(path)
	if err != nil {
		return st, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return st, nil
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, fmt.Errorf("parse fake state %s: %w", path, err)
	}
	return st, nil
}

func writeStateFile(path string, st fakeState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

// loadState reads the state named by GHS_FAKE_STATE (used inside the fakes).
func loadState() (fakeState, error) {
	return readStateFile(os.Getenv("GHS_FAKE_STATE"))
}

// saveState persists the state named by GHS_FAKE_STATE (used inside the fakes).
func saveState(st fakeState) error {
	return writeStateFile(os.Getenv("GHS_FAKE_STATE"), st)
}

// call is one invocation recorded in the log.
type call struct {
	Tool string   `json:"tool"`
	Args []string `json:"args"`
	Dir  string   `json:"dir,omitempty"`
	// Env lists the names (not values) of the environment variables the
	// fake saw, so isolation tests can prove the environment is minimal.
	Env []string `json:"env,omitempty"`
}

func (c call) String() string {
	return c.Tool + " " + strings.Join(c.Args, " ")
}

// appendLog writes one JSON line to GHS_FAKE_LOG before the fake does
// anything else.
func appendLog(tool string, args []string, dir string) error {
	path := os.Getenv("GHS_FAKE_LOG")
	if path == "" {
		return fmt.Errorf("GHS_FAKE_LOG is not set")
	}
	var env []string
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if name != "" {
			env = append(env, name)
		}
	}
	slices.Sort(env)
	if args == nil {
		args = []string{}
	}
	line, err := json.Marshal(call{Tool: tool, Args: args, Dir: dir, Env: env})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = f.Write(append(line, '\n'))
	return err
}

// failMatches reports the first pattern that is a substring of the
// space-joined argument vector.
func failMatches(patterns []string, args []string) (string, bool) {
	joined := strings.Join(args, " ")
	for _, p := range patterns {
		if p != "" && strings.Contains(joined, p) {
			return p, true
		}
	}
	return "", false
}

// fakeCtx is the per-invocation context shared by the fakes.
type fakeCtx struct {
	tool   string
	st     *fakeState
	dir    string
	stdout io.Writer
	stderr io.Writer
	dirty  bool
}

func (f *fakeCtx) unsupported(args []string) int {
	_, _ = fmt.Fprintf(f.stderr, "fake %s: unsupported arguments: %s\n", f.tool, strings.Join(args, " "))
	return 64
}

func (f *fakeCtx) failf(code int, format string, a ...any) int {
	_, _ = fmt.Fprintf(f.stderr, format+"\n", a...)
	return code
}

// runFake is the entry point when the test binary runs as a fake tool.
func runFake(tool string, args []string) int {
	dir, _ := os.Getwd()
	if err := appendLog(tool, args, dir); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fake %s: log: %v\n", tool, err)
		return 70
	}
	st, err := loadState()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fake %s: state: %v\n", tool, err)
		return 70
	}
	f := &fakeCtx{tool: tool, st: &st, dir: dir, stdout: os.Stdout, stderr: os.Stderr}

	var patterns []string
	switch tool {
	case "gh":
		patterns = st.GH.Fail
	case "git":
		patterns = st.Git.Fail
	}
	if p, ok := failMatches(patterns, args); ok {
		return f.failf(1, "fake %s: forced failure (%s)", tool, p)
	}

	var code int
	switch tool {
	case "gh":
		code = f.gh(args)
	case "git":
		code = f.git(args)
	case "ssh":
		code = f.ssh(args)
	case "ssh-keygen":
		code = f.keygen(args)
	default:
		code = f.unsupported(args)
	}
	if f.dirty {
		if err := saveState(st); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "fake %s: save state: %v\n", tool, err)
			return 70
		}
	}
	return code
}
