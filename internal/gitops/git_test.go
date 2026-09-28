package gitops

import "testing"

func TestParseShowOriginLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line               string
		scope, file, value string
		ok                 bool
	}{
		{"local\tfile:/r/.git/config\tIzzam", "local", "/r/.git/config", "Izzam", true},
		{"global\tfile:C:/Users/x/.gitconfig\tme@example.com\n", "global", "C:/Users/x/.gitconfig", "me@example.com", true},
		{"global\tfile:/cfg/gitconfig-work\tName\twith tab", "global", "/cfg/gitconfig-work", "Name\twith tab", true},
		{"command\tcommand line:\tx", "command", "command line:", "x", true},
		{"", "", "", "", false},
		{"no tabs here", "", "", "", false},
	}
	for _, tt := range tests {
		scope, file, value, ok := ParseShowOriginLine(tt.line)
		if scope != tt.scope || file != tt.file || value != tt.value || ok != tt.ok {
			t.Errorf("ParseShowOriginLine(%q) = %q, %q, %q, %v", tt.line, scope, file, value, ok)
		}
	}
}
