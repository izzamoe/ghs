package cli

import (
	"testing"

	"github.com/izzamoe/ghs/internal/config"
)

func resolveFixture() config.Config {
	return config.Config{Profiles: []config.Profile{
		{Name: "work", GitHubUser: "zamyb-work", GitEmail: "work@example.com", SSHHostAlias: "github-work"},
		{Name: "me", GitHubUser: "zamyb", GitEmail: "me@example.com", SSHHostAlias: "github-me"},
	}}
}

func TestResolveByLogin(t *testing.T) {
	t.Parallel()
	cfg := resolveFixture()
	if r := resolveLogin(cfg, "Zamyb-Work"); r != (resolution{Profile: "work", Kind: kindProfile}) {
		t.Fatalf("resolveLogin = %+v", r)
	}
	if r := resolveLogin(cfg, "stranger"); r.Kind != kindNone || r.Profile != "" {
		t.Fatalf("resolveLogin(stranger) = %+v", r)
	}
	if r := resolveLogin(cfg, ""); r.Kind != kindUnavailable {
		t.Fatalf("resolveLogin(\"\") = %+v", r)
	}
}

func TestResolveByEmail(t *testing.T) {
	t.Parallel()
	cfg := resolveFixture()
	if r := resolveEmail(cfg, "ME@example.com"); r != (resolution{Profile: "me", Kind: kindProfile}) {
		t.Fatalf("resolveEmail = %+v", r)
	}
	if r := resolveEmail(cfg, "nobody@example.com"); r.Kind != kindNone {
		t.Fatalf("resolveEmail(nobody) = %+v", r)
	}
	if r := resolveEmail(cfg, ""); r.Kind != kindUnavailable {
		t.Fatalf("resolveEmail(\"\") = %+v", r)
	}
}

func TestResolveByAlias(t *testing.T) {
	t.Parallel()
	cfg := resolveFixture()
	if r := resolveAlias(cfg, "GITHUB-ME"); r != (resolution{Profile: "me", Kind: kindProfile}) {
		t.Fatalf("resolveAlias = %+v", r)
	}
	if r := resolveAlias(cfg, "github-x"); r.Kind != kindUnknownAlias {
		t.Fatalf("resolveAlias(unknown) = %+v", r)
	}
}

func TestResolveOrigin(t *testing.T) {
	t.Parallel()
	cfg := resolveFixture()
	tests := []struct {
		url     string
		kind    string
		profile string
		host    string
	}{
		{"git@github-me:acme/app.git", kindProfile, "me", "github-me"},
		{"ssh://git@github-work/acme/app.git", kindProfile, "work", "github-work"},
		{"", kindNone, "", ""},
		{"git@github.com:acme/app.git", kindUnaliased, "", "github.com"},
		{"https://github.com/acme/app", kindUnaliased, "", "github.com"},
		{"https://gitlab.com/g/r.git", kindNotGitHub, "", "gitlab.com"},
		{"garbage", kindNotGitHub, "", ""},
		{"git@gitlab.com:g/r.git", kindUnknownAlias, "", "gitlab.com"},
		{"git@some-alias:o/r.git", kindUnknownAlias, "", "some-alias"},
	}
	for _, tt := range tests {
		r, remote := resolveOrigin(cfg, tt.url)
		if r.Kind != tt.kind || r.Profile != tt.profile || remote.Host != tt.host {
			t.Errorf("resolveOrigin(%q) = %+v host %q; want kind %s profile %q host %q", tt.url, r, remote.Host, tt.kind, tt.profile, tt.host)
		}
	}
}
