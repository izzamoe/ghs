package config

import (
	"strings"
	"testing"
)

var reservedForTest = []string{"add-profile", "list", "use", "help", "version"}

func checkValidator(t *testing.T, name string, fn func(string) error, accept, reject []string) {
	t.Helper()
	for _, v := range accept {
		if err := fn(v); err != nil {
			t.Errorf("%s(%q) = %v, want nil", name, v, err)
		}
	}
	for _, v := range reject {
		if err := fn(v); err == nil {
			t.Errorf("%s(%q) = nil, want error", name, v)
		}
	}
}

func TestValidateName(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateName", func(s string) error { return ValidateName(s, reservedForTest) },
		[]string{"work", "Work", "me.2", "a_b-c", "işçi", "工作", strings.Repeat("a", 64)},
		[]string{"", "wo]rk", "[work", "my work", "a/b", "a\\b", "a\x01b", "a\nb", "-work", ".work", "help", "HELP", "list", "--help", strings.Repeat("a", 65), "a\"b", "a#b", "a=b"},
	)
}

func TestValidateAlias(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateAlias", ValidateAlias,
		[]string{"github-work", "gh.work_2", strings.Repeat("a", 64)},
		[]string{"", "github work", "a#b", "a\"b", "a\tb", "a\x00b", "-alias", "github.com", "GitHub.com", "a*b", "a?b", "a/b", strings.Repeat("a", 65), "héllo"},
	)
}

func TestValidateLogin(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateLogin", ValidateLogin,
		[]string{"zamyb", "zamyb-work", "A1", strings.Repeat("a", 39)},
		[]string{"", "-bad", "bad-", "-bad-", "a--b", "a_b", "a b", "bad\nname", strings.Repeat("a", 40), "a\"b"},
	)
}

func TestValidateGitName(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateGitName", ValidateGitName,
		[]string{"Izzam", "IZZAMUDDIN ROYHUL FIRDAUS", "O'Brien", "Zoë #1"},
		[]string{"", "   ", "a\"b", "a\nb", "a\rb", "a\x7fb"},
	)
}

func TestValidateEmail(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateEmail", ValidateEmail,
		[]string{"work@example.com", "275592473+zamyb@users.noreply.github.com", "a.b@x.co"},
		[]string{"", "a@b", "@x.com", "a b@x.com", "a@x.com\n", "a@@x.com", "a@b@x.com", "a\"b@x.com", "a@.com", "a@x.", "a@x..com", "plain", "a@x .com", "a\t@x.com"},
	)
}

func TestValidateKeyPath(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateKeyPath", ValidateKeyPath,
		[]string{"~/.ssh/id_ed25519_work", "/home/u/.ssh/id", "~/My Keys/id", "C:/Users/x/.ssh/id", `C:\Users\x\.ssh\id`},
		[]string{"", "relative/key", "id_ed25519", "~user/key", "~", "/a\nb", "~/a\"b", "~/a\x01"},
	)
}

func TestValidateWorkspace(t *testing.T) {
	t.Parallel()
	checkValidator(t, "ValidateWorkspace", ValidateWorkspace,
		[]string{"~/Documents/work", "/srv/work", "~/Documents/work/", `C:\Users\x\work`, "C:/Users/x/work"},
		[]string{"", "relative", "~", "~/", "/", "C:/", `C:\`, "~/a\nb", "~/a\"b", "~/a\x01", "~user/x", "//"},
	)
}

func TestNormalizeWorkspace(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"~/Documents/work":  "~/Documents/work",
		"~/Documents/work/": "~/Documents/work",
		`C:\Users\x\work\`:  "C:/Users/x/work",
		"/srv/work//":       "/srv/work",
	} {
		if got := NormalizeWorkspace(in); got != want {
			t.Errorf("NormalizeWorkspace(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSafeSuffix(t *testing.T) {
	t.Parallel()
	for in, want := range map[string]string{
		"Work":     "work",
		"zamyb":    "zamyb",
		"a.b_c-d":  "a.b_c-d",
		"工作":       "profile",
		"işçi-dev": "i-i-dev",
	} {
		if got := SafeSuffix(in); got != want {
			t.Errorf("SafeSuffix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckUnique(t *testing.T) {
	t.Parallel()

	cfg := Config{Profiles: []Profile{
		{Name: "work", SSHHostAlias: "github-work", Workspace: "~/Documents/work"},
		{Name: "me", SSHHostAlias: "github-me"},
	}}
	tests := []struct {
		name    string
		p       Profile
		ignore  int
		wantErr string
	}{
		{name: "new profile ok", p: Profile{Name: "old", SSHHostAlias: "github-old", Workspace: "~/code"}, ignore: -1},
		{name: "same index ignored", p: Profile{Name: "work", SSHHostAlias: "github-work", Workspace: "~/Documents/work"}, ignore: 0},
		{name: "name differs by case", p: Profile{Name: "Work", SSHHostAlias: "github-x"}, ignore: -1, wantErr: `"work"`},
		{name: "exact name without ignore", p: Profile{Name: "me", SSHHostAlias: "github-x"}, ignore: -1, wantErr: `"me"`},
		{name: "alias case-insensitive", p: Profile{Name: "old", SSHHostAlias: "GitHub-Me"}, ignore: -1, wantErr: `profile "me"`},
		{name: "workspace equal", p: Profile{Name: "old", SSHHostAlias: "github-old", Workspace: "~/Documents/work/"}, ignore: -1, wantErr: `profile "work"`},
		{name: "workspace nested child", p: Profile{Name: "old", SSHHostAlias: "github-old", Workspace: "~/Documents/work/sub"}, ignore: -1, wantErr: `profile "work"`},
		{name: "workspace nested parent", p: Profile{Name: "old", SSHHostAlias: "github-old", Workspace: "~/Documents"}, ignore: -1, wantErr: `profile "work"`},
		{name: "workspace sibling prefix ok", p: Profile{Name: "old", SSHHostAlias: "github-old", Workspace: "~/Documents/workshop"}, ignore: -1},
		{name: "workspace case-insensitive", p: Profile{Name: "old", SSHHostAlias: "github-old", Workspace: "~/documents/WORK"}, ignore: -1, wantErr: `profile "work"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := cfg.CheckUnique(tt.p, tt.ignore)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("CheckUnique() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("CheckUnique() = %v, want error containing %s", err, tt.wantErr)
			}
		})
	}
}
