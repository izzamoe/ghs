package ghops

import "testing"

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
