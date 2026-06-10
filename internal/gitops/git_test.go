package gitops

import "testing"

func TestRewriteGitHubURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "ssh scp syntax",
			url:  "git@github.com:owner/repo.git",
			want: "git@github-work:owner/repo.git",
		},
		{
			name: "ssh url syntax",
			url:  "ssh://git@github.com/owner/repo.git",
			want: "ssh://git@github-work/owner/repo.git",
		},
		{
			name: "https syntax",
			url:  "https://github.com/owner/repo.git",
			want: "git@github-work:owner/repo.git",
		},
		{
			name: "already rewritten",
			url:  "git@github-work:owner/repo.git",
			want: "git@github-work:owner/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := RewriteGitHubURL(tt.url, "github-work")
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

	if _, err := RewriteGitHubURL("https://gitlab.com/owner/repo.git", "github-work"); err == nil {
		t.Fatal("RewriteGitHubURL() error = nil, want error")
	}
}

func TestCloneURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "owner repo",
			input: "itybtech/yb-trading-fundamental-services",
			want:  "git@github-work:itybtech/yb-trading-fundamental-services.git",
		},
		{
			name:  "https url",
			input: "https://github.com/itybtech/yb-trading-fundamental-services.git",
			want:  "git@github-work:itybtech/yb-trading-fundamental-services.git",
		},
		{
			name:  "http url",
			input: "http://github.com/itybtech/yb-trading-fundamental-services.git",
			want:  "git@github-work:itybtech/yb-trading-fundamental-services.git",
		},
		{
			name:  "ssh url",
			input: "ssh://git@github.com/itybtech/yb-trading-fundamental-services.git",
			want:  "ssh://git@github-work/itybtech/yb-trading-fundamental-services.git",
		},
		{
			name:  "scp ssh url",
			input: "git@github.com:itybtech/yb-trading-fundamental-services.git",
			want:  "git@github-work:itybtech/yb-trading-fundamental-services.git",
		},
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

func TestCloneDirectory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "itybtech/yb-trading-fundamental-services", want: "yb-trading-fundamental-services"},
		{input: "https://github.com/itybtech/yb-trading-fundamental-services", want: "yb-trading-fundamental-services"},
		{input: "https://github.com/itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "http://github.com/itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "ssh://git@github.com/itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "git@github.com:itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
		{input: "git@github-work:itybtech/yb-trading-fundamental-services.git", want: "yb-trading-fundamental-services"},
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
