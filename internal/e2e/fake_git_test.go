package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// git implements the "Fake git" table of contracts/fake-tools.md.
func (f *fakeCtx) git(args []string) int {
	dir := f.dir
	rest := args
	if len(rest) >= 2 && rest[0] == "-C" {
		dir = rest[1]
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(f.dir, dir)
		}
		rest = rest[2:]
	}
	if len(rest) == 0 {
		return f.unsupported(args)
	}
	g := &f.st.Git
	switch rest[0] {
	case "rev-parse":
		if len(rest) == 2 && rest[1] == "--show-toplevel" {
			if !g.Repo {
				return f.notRepo()
			}
			fmt.Fprintln(f.stdout, g.Toplevel)
			return 0
		}
	case "remote":
		if len(rest) == 3 && rest[1] == "get-url" && rest[2] == "origin" {
			if !g.Repo {
				return f.notRepo()
			}
			if g.Origin == "" {
				return f.failf(2, "error: No such remote 'origin'")
			}
			fmt.Fprintln(f.stdout, g.Origin)
			return 0
		}
		if len(rest) == 4 && rest[1] == "set-url" && rest[2] == "origin" {
			if !g.Repo {
				return f.notRepo()
			}
			if g.Origin == "" {
				return f.failf(2, "error: No such remote 'origin'")
			}
			g.Origin = rest[3]
			f.dirty = true
			return 0
		}
	case "config":
		return f.gitConfig(args, rest[1:], dir)
	case "clone":
		if len(rest) == 3 {
			return f.gitClone(rest[1], rest[2])
		}
	}
	return f.unsupported(args)
}

func (f *fakeCtx) notRepo() int {
	return f.failf(128, "fatal: not a git repository (or any of the parent directories): .git")
}

func (f *fakeCtx) gitConfig(all, args []string, dir string) int {
	var global, showOrigin, showScope, get, getAll, add, unset, fixedValue bool
	var pos []string
	for _, a := range args {
		switch a {
		case "--global":
			global = true
		case "--show-origin":
			showOrigin = true
		case "--show-scope":
			showScope = true
		case "--get":
			get = true
		case "--get-all":
			getAll = true
		case "--add":
			add = true
		case "--unset":
			unset = true
		case "--fixed-value":
			fixedValue = true
		default:
			if strings.HasPrefix(a, "-") {
				return f.unsupported(all)
			}
			pos = append(pos, a)
		}
	}
	g := &f.st.Git
	switch {
	case get && showOrigin && showScope && !getAll && !add && !unset && len(pos) == 1:
		scope, origin, value, ok := f.resolveConfig(pos[0], global, dir)
		if !ok {
			return 1
		}
		fmt.Fprintf(f.stdout, "%s\t%s\t%s\n", scope, origin, value)
		return 0

	case getAll && global && !get && !add && !unset && !showOrigin && !showScope && len(pos) == 1:
		values := g.Global[pos[0]]
		if len(values) == 0 {
			return 1
		}
		for _, v := range values {
			fmt.Fprintln(f.stdout, v)
		}
		return 0

	case add && global && !get && !getAll && !unset && !showOrigin && !showScope && len(pos) == 2:
		if g.Global == nil {
			g.Global = map[string]gitValues{}
		}
		g.Global[pos[0]] = append(g.Global[pos[0]], pos[1])
		f.dirty = true
		return 0

	case unset && fixedValue && global && !get && !getAll && !add && !showOrigin && !showScope && len(pos) == 2:
		values := g.Global[pos[0]]
		idx := slices.Index(values, pos[1])
		if idx < 0 {
			return 5
		}
		values = slices.Delete(slices.Clone(values), idx, idx+1)
		if len(values) == 0 {
			delete(g.Global, pos[0])
		} else {
			g.Global[pos[0]] = values
		}
		f.dirty = true
		return 0

	case !get && !getAll && !add && !unset && !fixedValue && !showOrigin && !showScope && len(pos) == 2:
		if pos[0] != "user.name" && pos[0] != "user.email" {
			return f.unsupported(all)
		}
		if global {
			if g.Global == nil {
				g.Global = map[string]gitValues{}
			}
			g.Global[pos[0]] = gitValues{pos[1]}
		} else {
			if !g.Repo {
				return f.failf(128, "fatal: not in a git directory")
			}
			if g.Local == nil {
				g.Local = map[string]gitValues{}
			}
			g.Local[pos[0]] = gitValues{pos[1]}
		}
		f.dirty = true
		return 0
	}
	return f.unsupported(all)
}

// resolveConfig resolves key the way real Git would for the scenarios the
// suite models: local, then a matching includeIf.gitdir/i identity file, then
// global.
func (f *fakeCtx) resolveConfig(key string, global bool, dir string) (scope, origin, value string, ok bool) {
	g := f.st.Git
	if !global && g.Repo {
		if v := g.Local[key]; len(v) > 0 {
			return "local", "file:" + filepath.ToSlash(g.Toplevel) + "/.git/config", v[len(v)-1], true
		}
	}
	if !global && g.Repo {
		home := os.Getenv("HOME")
		cwd := strings.ToLower(filepath.ToSlash(dir)) + "/"
		keys := make([]string, 0, len(g.Global))
		for k := range g.Global {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		var found bool
		for _, k := range keys {
			pattern, isInclude := strings.CutPrefix(k, "includeIf.gitdir/i:")
			pattern, hasSuffix := strings.CutSuffix(pattern, ".path")
			if !isInclude || !hasSuffix {
				continue
			}
			if rest, ok := strings.CutPrefix(pattern, "~/"); ok {
				pattern = filepath.ToSlash(home) + "/" + rest
			}
			if !strings.HasPrefix(cwd, strings.ToLower(pattern)) {
				continue
			}
			for _, file := range g.Global[k] {
				if v, ok := readIdentityValue(file, key); ok {
					scope, origin, value, found = "global", "file:"+filepath.ToSlash(file), v, true
				}
			}
		}
		if found {
			return scope, origin, value, true
		}
	}
	if v := g.Global[key]; len(v) > 0 {
		return "global", "file:" + filepath.ToSlash(os.Getenv("HOME")) + "/.gitconfig", v[len(v)-1], true
	}
	return "", "", "", false
}

// readIdentityValue reads user.name or user.email from a ghs identity file.
func readIdentityValue(path, key string) (string, bool) {
	section, name, ok := strings.Cut(key, ".")
	if !ok {
		return "", false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	var current, value string
	var found bool
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			current = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok || current != section || strings.TrimSpace(k) != name {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
			v = strings.ReplaceAll(v[1:len(v)-1], `\\`, `\`)
		}
		value, found = v, true
	}
	return value, found
}

func (f *fakeCtx) gitClone(url, dest string) int {
	if !filepath.IsAbs(dest) {
		dest = filepath.Join(f.dir, dest)
	}
	if entries, err := os.ReadDir(dest); err == nil && len(entries) > 0 {
		return f.failf(128, "fatal: destination path '%s' already exists and is not an empty directory.", dest)
	}
	if err := os.MkdirAll(filepath.Join(dest, ".git"), 0o755); err != nil {
		return f.failf(128, "fatal: %v", err)
	}
	content := fmt.Sprintf("[remote \"origin\"]\n\turl = %s\n", url)
	if err := os.WriteFile(filepath.Join(dest, ".git", "config"), []byte(content), 0o644); err != nil {
		return f.failf(128, "fatal: %v", err)
	}
	fmt.Fprintf(f.stderr, "Cloning into '%s'...\n", dest)
	g := &f.st.Git
	g.Repo = true
	g.Toplevel = dest
	g.Origin = url
	g.Local = nil
	f.dirty = true
	return 0
}
