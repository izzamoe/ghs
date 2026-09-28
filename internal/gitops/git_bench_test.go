package gitops

import "testing"

var sinkString string
var sinkErr error

func BenchmarkRewriteGitHubURL_SCP(b *testing.B) {
	for b.Loop() {
		sinkString, sinkErr = RewriteGitHubURL("git@github.com:owner/repo.git", "github-work", nil)
	}
}

func BenchmarkRewriteGitHubURL_HTTPS(b *testing.B) {
	for b.Loop() {
		sinkString, sinkErr = RewriteGitHubURL("https://github.com/owner/repo.git", "github-work", nil)
	}
}

func BenchmarkRewriteGitHubURL_OtherAlias(b *testing.B) {
	for b.Loop() {
		sinkString, sinkErr = RewriteGitHubURL("git@github-other:owner/repo.git", "github-work", []string{"github-other"})
	}
}

func BenchmarkCloneDirectory(b *testing.B) {
	for b.Loop() {
		sinkString = CloneDirectory("https://github.com/owner/repo.git")
	}
}

func BenchmarkCloneURL(b *testing.B) {
	for b.Loop() {
		sinkString, sinkErr = CloneURL("owner/repo", "github-work")
	}
}
