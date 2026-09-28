package ghops

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestParseAuthAccounts(t *testing.T) {
	t.Parallel()

	accounts, err := ParseAuthAccounts("github.com", []byte(`{"hosts":{"github.com":[{"state":"success","active":true,"host":"github.com","login":"zamyb"},{"state":"success","active":false,"host":"github.com","login":"izzamoe"}]}}`))
	if err != nil {
		t.Fatalf("ParseAuthAccounts() error = %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("len(accounts) = %d, want 2", len(accounts))
	}
	if !accounts[0].Active || accounts[0].Login != "zamyb" || accounts[1].Login != "izzamoe" {
		t.Fatalf("accounts = %+v", accounts)
	}
}

func TestSelectPrimaryEmail(t *testing.T) {
	t.Parallel()

	email := SelectPrimaryEmail([]Email{
		{Email: "unverified@example.com", Primary: true, Verified: false},
		{Email: "secondary@example.com", Primary: false, Verified: true},
		{Email: "primary@example.com", Primary: true, Verified: true},
	})
	if email != "primary@example.com" {
		t.Fatalf("SelectPrimaryEmail() = %q, want primary@example.com", email)
	}
}

func TestSelectPrimaryEmailFallsBackToVerified(t *testing.T) {
	t.Parallel()

	email := SelectPrimaryEmail([]Email{
		{Email: "unverified@example.com", Primary: true, Verified: false},
		{Email: "verified@example.com", Primary: false, Verified: true},
	})
	if email != "verified@example.com" {
		t.Fatalf("SelectPrimaryEmail() = %q, want verified@example.com", email)
	}
}

func TestSelectPrimaryEmailFallsBackToAnyEmail(t *testing.T) {
	t.Parallel()

	email := SelectPrimaryEmail([]Email{{Email: "fallback@example.com"}})
	if email != "fallback@example.com" {
		t.Fatalf("SelectPrimaryEmail() = %q, want fallback@example.com", email)
	}
}

func TestSelectPrimaryEmailReturnsEmptyForNoEmail(t *testing.T) {
	t.Parallel()

	if email := SelectPrimaryEmail([]Email{}); email != "" {
		t.Fatalf("SelectPrimaryEmail() = %q, want empty", email)
	}
}

func TestNoreplyEmail(t *testing.T) {
	t.Parallel()

	email := NoreplyEmail(User{ID: 275592473, Login: "zamyb"})
	if email != "275592473+zamyb@users.noreply.github.com" {
		t.Fatalf("NoreplyEmail() = %q, want noreply email", email)
	}
}

// stubRunner answers commands from a table keyed by the space-joined
// argument vector (without the program name) and records every call.
type stubRunner struct {
	responses map[string]stubResponse
	calls     []string
}

type stubResponse struct {
	out  string
	code int
}

func (s *stubRunner) answer(name string, args []string) (string, int) {
	key := strings.Join(args, " ")
	s.calls = append(s.calls, name+" "+key)
	r, ok := s.responses[key]
	if !ok {
		return "", 0
	}
	return r.out, r.code
}

func (s *stubRunner) Run(name string, args ...string) error {
	out, code := s.answer(name, args)
	if code != 0 {
		return fmt.Errorf("run %s: exit status %d: %s", name, code, out)
	}
	return nil
}

func (s *stubRunner) OutputBytes(name string, args ...string) ([]byte, error) {
	out, code := s.answer(name, args)
	if code != 0 {
		return nil, fmt.Errorf("run %s: exit status %d: %s", name, code, out)
	}
	return []byte(out), nil
}

func (s *stubRunner) CombinedOutput(name string, args ...string) (string, int, error) {
	out, code := s.answer(name, args)
	return out, code, nil
}

const authStatusArgs = "auth status --hostname github.com --json hosts"

func statusJSON(accounts string) stubResponse {
	return stubResponse{out: `{"hosts":{"github.com":[` + accounts + `]}}`}
}

func TestActiveAccount(t *testing.T) {
	t.Parallel()

	run := &stubRunner{responses: map[string]stubResponse{authStatusArgs: statusJSON(
		`{"state":"success","active":false,"host":"github.com","login":"zamyb"},{"state":"success","active":true,"host":"github.com","login":"zamyb-work"}`)}}
	login, ok, err := New(run).ActiveAccount("github.com")
	if err != nil || !ok || login != "zamyb-work" {
		t.Fatalf("ActiveAccount() = %q, %v, %v", login, ok, err)
	}

	run = &stubRunner{responses: map[string]stubResponse{authStatusArgs: {out: `{"hosts":{}}`}}}
	login, ok, err = New(run).ActiveAccount("github.com")
	if err != nil || ok || login != "" {
		t.Fatalf("ActiveAccount(no accounts) = %q, %v, %v", login, ok, err)
	}
}

func TestIsAuthenticated(t *testing.T) {
	t.Parallel()

	run := &stubRunner{responses: map[string]stubResponse{authStatusArgs: statusJSON(
		`{"state":"success","active":true,"host":"github.com","login":"zamyb-work"},{"state":"error","active":false,"host":"github.com","login":"broken"}`)}}
	gh := New(run)
	for login, want := range map[string]bool{"zamyb-work": true, "Zamyb-Work": true, "broken": false, "absent": false} {
		got, err := gh.IsAuthenticated("github.com", login)
		if err != nil || got != want {
			t.Errorf("IsAuthenticated(%q) = %v, %v; want %v", login, got, err, want)
		}
	}
}

func TestWithAccount(t *testing.T) {
	t.Parallel()

	status := statusJSON(`{"state":"success","active":true,"host":"github.com","login":"zamyb"},{"state":"success","active":false,"host":"github.com","login":"zamyb-work"}`)

	// Already active: no switch at all.
	run := &stubRunner{responses: map[string]stubResponse{authStatusArgs: status}}
	switched, err := New(run).WithAccount("github.com", "zamyb", func() error { return nil })
	if err != nil || switched {
		t.Fatalf("WithAccount(active) = %v, %v", switched, err)
	}
	for _, c := range run.calls {
		if strings.Contains(c, "auth switch") {
			t.Fatalf("unexpected switch: %v", run.calls)
		}
	}

	// Different account: switch, run, switch back.
	run = &stubRunner{responses: map[string]stubResponse{authStatusArgs: status}}
	var during []string
	switched, err = New(run).WithAccount("github.com", "zamyb-work", func() error {
		during = append([]string(nil), run.calls...)
		return nil
	})
	if err != nil || !switched {
		t.Fatalf("WithAccount() = %v, %v", switched, err)
	}
	if last := during[len(during)-1]; last != "gh auth switch --hostname github.com --user zamyb-work" {
		t.Fatalf("fn ran after %q, want the switch to zamyb-work", last)
	}
	if last := run.calls[len(run.calls)-1]; last != "gh auth switch --hostname github.com --user zamyb" {
		t.Fatalf("last call = %q, want switch back to zamyb", last)
	}

	// fn fails: still switched back, and both results are reported.
	run = &stubRunner{responses: map[string]stubResponse{authStatusArgs: status}}
	_, err = New(run).WithAccount("github.com", "zamyb-work", func() error { return errors.New("upload exploded") })
	if err == nil || !strings.Contains(err.Error(), "upload exploded") || !strings.Contains(err.Error(), "restored gh account zamyb") {
		t.Fatalf("err = %v, want failure and restore result", err)
	}

	// fn fails and restore fails: both errors.
	run = &stubRunner{responses: map[string]stubResponse{
		authStatusArgs: status,
		"auth switch --hostname github.com --user zamyb": {out: "cannot switch back", code: 1},
	}}
	_, err = New(run).WithAccount("github.com", "zamyb-work", func() error { return errors.New("upload exploded") })
	if err == nil || !strings.Contains(err.Error(), "upload exploded") || !strings.Contains(err.Error(), "could not restore gh account zamyb") {
		t.Fatalf("err = %v, want failure and restore failure", err)
	}
}

func TestAddSSHKeyAlreadyPresentFromStderr(t *testing.T) {
	t.Parallel()

	args := "ssh-key add /k.pub --title ghs-work"
	for _, tc := range []struct {
		name    string
		resp    stubResponse
		already bool
		wantErr bool
	}{
		{"added", stubResponse{out: "✓ Public key added to your account"}, false, false},
		{"exists exit 0", stubResponse{out: "✓ Public key already exists on your account"}, true, false},
		{"already in use non-zero", stubResponse{out: "HTTP 422: Validation Failed (key is already in use)", code: 1}, true, false},
		{"other failure", stubResponse{out: "HTTP 401: Bad credentials", code: 1}, false, true},
	} {
		run := &stubRunner{responses: map[string]stubResponse{args: tc.resp}}
		already, err := New(run).AddSSHKey("/k.pub", "ghs-work")
		if already != tc.already || (err != nil) != tc.wantErr {
			t.Errorf("%s: AddSSHKey() = %v, %v", tc.name, already, err)
		}
	}
}
