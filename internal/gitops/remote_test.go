package gitops

import "testing"

func TestParseRemote(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		kind RemoteKind
		host string
		path string
	}{
		{"git@github.com:o/r.git", RemoteGitHub, "github.com", "o/r.git"},
		{"git@GitHub.com:o/r", RemoteGitHub, "github.com", "o/r.git"},
		{"ssh://git@github.com/o/r", RemoteGitHub, "github.com", "o/r.git"},
		{"ssh://git@github.com:22/o/r.git", RemoteGitHub, "github.com", "o/r.git"},
		{"https://github.com/o/r/", RemoteGitHub, "github.com", "o/r.git"},
		{"https://user@github.com/o/r.git", RemoteGitHub, "github.com", "o/r.git"},
		{"http://github.com/o/r.git", RemoteGitHub, "github.com", "o/r.git"},
		{"https://github.com/o/r.git.git/", RemoteGitHub, "github.com", "o/r.git"},
		{"git@github-work:o/r.git", RemoteSSHHost, "github-work", "o/r.git"},
		{"ssh://git@github-me/o/r.git", RemoteSSHHost, "github-me", "o/r.git"},
		{"git@gitlab.com:g/r.git", RemoteSSHHost, "gitlab.com", "g/r.git"},
		{"https://gitlab.com/g/r", RemoteOther, "gitlab.com", "g/r.git"},
		{"https://ghe.example.com/o/r.git", RemoteOther, "ghe.example.com", "o/r.git"},
		{"garbage", RemoteOther, "", ""},
		{"git@github.com:", RemoteOther, "", ""},
		{"https://github.com/", RemoteOther, "github.com", ""},
		{"/local/path/repo.git", RemoteOther, "", ""},
		{"file:///local/repo.git", RemoteOther, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			r, err := ParseRemote(tt.url)
			if err != nil {
				t.Fatalf("ParseRemote() error = %v", err)
			}
			if r.Kind != tt.kind || r.Host != tt.host || r.Path != tt.path || r.Raw != tt.url {
				t.Fatalf("ParseRemote() = %+v, want kind %v host %q path %q", r, tt.kind, tt.host, tt.path)
			}
		})
	}
	if _, err := ParseRemote(""); err == nil {
		t.Fatal("ParseRemote(\"\") error = nil")
	}
}

func TestAliasURL(t *testing.T) {
	t.Parallel()

	r, _ := ParseRemote("https://github.com/owner/repo/")
	if got := r.AliasURL("github-work"); got != "git@github-work:owner/repo.git" {
		t.Fatalf("AliasURL() = %q", got)
	}
}

func TestRewriteGitHubURL(t *testing.T) {
	t.Parallel()

	aliases := []string{"github-work", "github-other"}
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "ssh scp syntax", url: "git@github.com:owner/repo.git", want: "git@github-work:owner/repo.git"},
		{name: "ssh url syntax", url: "ssh://git@github.com/owner/repo.git", want: "git@github-work:owner/repo.git"},
		{name: "https syntax", url: "https://github.com/owner/repo.git", want: "git@github-work:owner/repo.git"},
		{name: "https without suffix", url: "https://github.com/owner/repo", want: "git@github-work:owner/repo.git"},
		{name: "http trailing slash", url: "http://github.com/owner/repo/", want: "git@github-work:owner/repo.git"},
		{name: "already rewritten", url: "git@github-work:owner/repo.git", want: "git@github-work:owner/repo.git"},
		{name: "aliased from different profile", url: "git@github-other:owner/repo.git", want: "git@github-work:owner/repo.git"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RewriteGitHubURL(tt.url, "github-work", aliases)
			if err != nil {
				t.Fatalf("RewriteGitHubURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("RewriteGitHubURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRewriteGitHubURLRejectsUnsupportedURL(t *testing.T) {
	t.Parallel()

	for _, url := range []string{
		"https://gitlab.com/owner/repo.git",
		"git@gitlab.com:owner/repo.git",
		"git@some-alias:owner/repo.git",
		"ssh://git@ghe.example.com/owner/repo.git",
		"garbage",
	} {
		if _, err := RewriteGitHubURL(url, "github-work", []string{"github-work"}); err == nil {
			t.Errorf("RewriteGitHubURL(%q) error = nil, want error", url)
		}
	}
	if _, err := RewriteGitHubURL("git@github.com:o/r.git", "", nil); err == nil {
		t.Error("empty alias accepted")
	}
}

func TestCloneURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "owner repo", input: "itybtech/yb-trading-fundamental-services", want: "git@github-work:itybtech/yb-trading-fundamental-services.git"},
		{name: "owner repo with suffix", input: "o/r.git", want: "git@github-work:o/r.git"},
		{name: "https url", input: "https://github.com/itybtech/yb-trading-fundamental-services.git", want: "git@github-work:itybtech/yb-trading-fundamental-services.git"},
		{name: "http url", input: "http://github.com/itybtech/yb-trading-fundamental-services.git", want: "git@github-work:itybtech/yb-trading-fundamental-services.git"},
		{name: "ssh url", input: "ssh://git@github.com/itybtech/yb-trading-fundamental-services.git", want: "git@github-work:itybtech/yb-trading-fundamental-services.git"},
		{name: "scp ssh url", input: "git@github.com:itybtech/yb-trading-fundamental-services.git", want: "git@github-work:itybtech/yb-trading-fundamental-services.git"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := CloneURL(tt.input, "github-work")
			if err != nil {
				t.Fatalf("CloneURL() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("CloneURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCloneURLRejectsNonGitHub(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"https://gitlab.com/g/r.git", "git@gitlab.com:g/r.git", "git@github-other:o/r.git", "a/b/c", "garbage"} {
		if _, err := CloneURL(input, "github-work"); err == nil {
			t.Errorf("CloneURL(%q) error = nil", input)
		}
	}
}

func TestCloneDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "itybtech/yb-trading-fundamental-services", want: "yb-trading-fundamental-services"},
		{input: "https://github.com/itybtech/yb-trading-fundamental-services", want: "yb-trading-fundamental-services"},
		{input: "https://github.com/itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "https://github.com/itybtech/yb-trading-fundamental-services/", want: "yb-trading-fundamental-services"},
		{input: "http://github.com/itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "ssh://git@github.com/itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "git@github.com:itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "git@github-work:itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "garbage", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := CloneDirectory(tt.input); got != tt.want {
				t.Fatalf("CloneDirectory() = %q, want %q", got, tt.want)
			}
		})
	}
}
