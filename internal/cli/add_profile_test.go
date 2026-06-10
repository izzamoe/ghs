package cli

import (
	"testing"

	"github.com/izzamoe/ghs/internal/ghops"
)

func TestProfileFromGHUsesDefaults(t *testing.T) {
	t.Parallel()

	profile, err := profileFromGH("Work Account", ghops.User{
		Login: "zamyb",
		Name:  "IZZAMUDDIN",
		Email: "work@example.com",
	}, map[string]string{})
	if err != nil {
		t.Fatalf("profileFromGH() error = %v", err)
	}

	if profile.GitHubUser != "zamyb" {
		t.Fatalf("GitHubUser = %q, want zamyb", profile.GitHubUser)
	}
	if profile.GitName != "IZZAMUDDIN" {
		t.Fatalf("GitName = %q, want IZZAMUDDIN", profile.GitName)
	}
	if profile.GitEmail != "work@example.com" {
		t.Fatalf("GitEmail = %q, want work@example.com", profile.GitEmail)
	}
	if profile.SSHHostAlias != "github-work-account" {
		t.Fatalf("SSHHostAlias = %q, want github-work-account", profile.SSHHostAlias)
	}
	if profile.SSHKey != "~/.ssh/id_ed25519_work-account" {
		t.Fatalf("SSHKey = %q, want ~/.ssh/id_ed25519_work-account", profile.SSHKey)
	}
}

func TestProfileFromGHRequiresPrivateEmailOverride(t *testing.T) {
	t.Parallel()

	// user has no ID so NoreplyEmail returns "" — require-email must still error
	_, err := profileFromGH("work", ghops.User{Login: "zamyb"}, map[string]string{"require-email": ""})
	if err == nil {
		t.Fatal("profileFromGH() error = nil, want error")
	}
}

func TestProfileFromGHUsesNoreplyEmailFallback(t *testing.T) {
	t.Parallel()

	profile, err := profileFromGH("work", ghops.User{Login: "zamyb", ID: 275592473}, map[string]string{})
	if err != nil {
		t.Fatalf("profileFromGH() error = %v", err)
	}
	if profile.GitEmail != "275592473+zamyb@users.noreply.github.com" {
		t.Fatalf("GitEmail = %q, want noreply email", profile.GitEmail)
	}
}

func TestProfileFromGHAllowsMissingEmailWhenNoID(t *testing.T) {
	t.Parallel()

	profile, err := profileFromGH("work", ghops.User{Login: "zamyb"}, map[string]string{})
	if err != nil {
		t.Fatalf("profileFromGH() error = %v", err)
	}
	if profile.GitEmail != "" {
		t.Fatalf("GitEmail = %q, want empty", profile.GitEmail)
	}
}

func TestProfileFromGHAllowsOverrides(t *testing.T) {
	t.Parallel()

	profile, err := profileFromGH("work", ghops.User{Login: "zamyb"}, map[string]string{
		"git-name":  "Work Name",
		"git-email": "work@example.com",
		"ssh-alias": "github-zamyb",
		"ssh-key":   "~/.ssh/id_ed25519_zamyb",
		"workspace": "~/Documents/work",
	})
	if err != nil {
		t.Fatalf("profileFromGH() error = %v", err)
	}

	if profile.GitName != "Work Name" || profile.SSHHostAlias != "github-zamyb" || profile.Workspace != "~/Documents/work" {
		t.Fatalf("profile = %+v", profile)
	}
}

func TestActiveAccountLogin(t *testing.T) {
	t.Parallel()

	login := activeAccountLogin([]ghops.AuthAccount{
		{Login: "inactive", Active: false},
		{Login: "active", Active: true},
	})
	if login != "active" {
		t.Fatalf("activeAccountLogin() = %q, want active", login)
	}
}
