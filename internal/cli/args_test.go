package cli

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	t.Parallel()

	use := cmdSpec{Name: "use", MinPos: 1, MaxPos: 1, Flags: []flagSpec{
		{Name: "global"}, {Name: "fix-remote"},
	}}
	add := cmdSpec{Name: "add-profile", MinPos: 1, MaxPos: 1, Flags: []flagSpec{
		{Name: "gh-user", TakesValue: true}, {Name: "git-email", TakesValue: true},
	}}
	none := cmdSpec{Name: "list"}

	tests := []struct {
		name      string
		spec      cmdSpec
		args      []string
		wantPos   []string
		wantFlags map[string]string
		wantErr   string // substring of the UsageError message; "" = no error
		wantHelp  bool
	}{
		{name: "unknown flag", spec: use, args: []string{"work", "--globl"}, wantErr: "unknown flag --globl"},
		{name: "unknown single dash flag", spec: use, args: []string{"work", "-g"}, wantErr: "unknown flag -g"},
		{name: "unknown flag on flagless command", spec: none, args: []string{"--all"}, wantErr: "unknown flag --all"},
		{name: "missing value at end", spec: add, args: []string{"work", "--gh-user"}, wantErr: "flag --gh-user requires a value"},
		{name: "missing value before another flag", spec: add, args: []string{"work", "--gh-user", "--git-email", "a@x.com"}, wantErr: "flag --gh-user requires a value"},
		{name: "empty value", spec: add, args: []string{"work", "--gh-user", ""}, wantErr: "flag --gh-user requires a value"},
		{name: "empty equals value", spec: add, args: []string{"work", "--gh-user="}, wantErr: "flag --gh-user requires a value"},
		{name: "duplicate flag", spec: use, args: []string{"work", "--global", "--global"}, wantErr: "flag --global given more than once"},
		{name: "duplicate value flag", spec: add, args: []string{"w", "--gh-user", "a", "--gh-user", "b"}, wantErr: "flag --gh-user given more than once"},
		{name: "value on boolean flag", spec: use, args: []string{"work", "--global", "yes"}, wantErr: "flag --global does not take a value"},
		{name: "equals value on boolean flag", spec: use, args: []string{"work", "--global=yes"}, wantErr: "flag --global does not take a value"},
		{name: "surplus positional", spec: use, args: []string{"work", "extra"}, wantErr: `unexpected argument "extra"`},
		{name: "surplus positional on list", spec: none, args: []string{"extra"}, wantErr: `unexpected argument "extra"`},
		{name: "too few positionals", spec: use, args: nil, wantErr: "missing required argument"},
		{
			name: "flags before positionals", spec: use, args: []string{"--global", "work"},
			wantPos: []string{"work"}, wantFlags: map[string]string{"global": ""},
		},
		{
			name: "value flags anywhere", spec: add, args: []string{"--gh-user", "zamyb", "work", "--git-email=a@x.com"},
			wantPos: []string{"work"}, wantFlags: map[string]string{"gh-user": "zamyb", "git-email": "a@x.com"},
		},
		{
			name: "double dash ends flags", spec: use, args: []string{"--", "--global"},
			wantPos: []string{"--global"}, wantFlags: map[string]string{},
		},
		{
			name: "double dash after flags", spec: use, args: []string{"--global", "--", "-work"},
			wantPos: []string{"-work"}, wantFlags: map[string]string{"global": ""},
		},
		{name: "long help", spec: use, args: []string{"--help"}, wantHelp: true},
		{name: "short help after positional", spec: use, args: []string{"work", "-h"}, wantHelp: true},
		{name: "help wins over unknown flag", spec: use, args: []string{"--bogus", "--help"}, wantHelp: true},
		{
			name: "help after double dash is positional", spec: use, args: []string{"--", "--help"},
			wantPos: []string{"--help"}, wantFlags: map[string]string{},
		},
		{
			name: "single dash is positional", spec: use, args: []string{"-"},
			wantPos: []string{"-"}, wantFlags: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pos, flags, err := parseArgs(tt.spec, tt.args)
			if tt.wantHelp {
				if !errors.Is(err, errHelp) {
					t.Fatalf("err = %v, want errHelp", err)
				}
				return
			}
			if tt.wantErr != "" {
				var usageErr *UsageError
				if !errors.As(err, &usageErr) {
					t.Fatalf("err = %v (%T), want *UsageError", err, err)
				}
				if !strings.Contains(usageErr.Message, tt.wantErr) {
					t.Fatalf("message = %q, want substring %q", usageErr.Message, tt.wantErr)
				}
				if usageErr.Command != tt.spec.Name {
					t.Fatalf("Command = %q, want %q", usageErr.Command, tt.spec.Name)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if !slices.Equal(pos, tt.wantPos) {
				t.Fatalf("pos = %q, want %q", pos, tt.wantPos)
			}
			if !maps.Equal(flags, tt.wantFlags) {
				t.Fatalf("flags = %v, want %v", flags, tt.wantFlags)
			}
		})
	}
}

func TestUsageErrorListsAcceptedFlags(t *testing.T) {
	t.Parallel()

	spec, ok := lookupCommand("use")
	if !ok {
		t.Fatal("use is not registered")
	}
	_, _, err := parseArgs(spec, []string{"work", "--globl"})
	var usageErr *UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("err = %v, want *UsageError", err)
	}
	for _, want := range []string{"ghs use <profile> [--global] [--fix-remote]", "--global", "--fix-remote"} {
		if !strings.Contains(usageErr.Usage, want) {
			t.Fatalf("usage %q does not mention %q", usageErr.Usage, want)
		}
	}
}
