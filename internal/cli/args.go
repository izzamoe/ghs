package cli

import (
	"errors"
	"strings"
)

// flagSpec declares one flag of a command.
type flagSpec struct {
	Name       string
	TakesValue bool
	// Value is the placeholder shown in usage text for value flags.
	Value string
	Help  string
}

// cmdSpec declares a command's grammar. It is pure data; dispatch lives in
// App.Run.
type cmdSpec struct {
	Name string
	// Usage holds the command's line(s) from the command list, verbatim.
	Usage  []string
	MinPos int
	MaxPos int
	Flags  []flagSpec
}

// errHelp is returned by parseArgs when --help or -h is present.
var errHelp = errors.New("help requested")

// parseArgs strictly parses args for spec. Flags may appear anywhere; "--"
// ends flag parsing. Unknown flags, missing or empty values, duplicates,
// values on boolean flags, and too few or too many positionals are usage
// errors. Parsing is pure, so no tool can run before a usage error.
func parseArgs(spec cmdSpec, args []string) (pos []string, flags map[string]string, err error) {
	for _, arg := range args {
		if arg == "--" {
			break
		}
		if arg == "--help" || arg == "-h" {
			return nil, nil, errHelp
		}
	}

	flags = map[string]string{}
	pos = []string{}
	// afterBool records positionals that directly follow a boolean flag, so
	// "--global yes" can be reported as a value on a boolean flag.
	afterBool := map[int]string{}
	lastBool := ""
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			if lastBool != "" {
				afterBool[len(pos)] = lastBool
			}
			pos = append(pos, arg)
			lastBool = ""
			continue
		}
		lastBool = ""

		name, value, hasValue := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		display := arg
		if hasValue {
			display, _, _ = strings.Cut(arg, "=")
		}
		f, ok := spec.flag(name)
		if !ok || !strings.HasPrefix(arg, "--") {
			return nil, nil, usageErrorf(spec.Name, "unknown flag %s for ghs %s%s", display, spec.Name, spec.acceptedFlags())
		}
		if _, dup := flags[f.Name]; dup {
			return nil, nil, usageErrorf(spec.Name, "flag --%s given more than once", f.Name)
		}
		if !f.TakesValue {
			if hasValue {
				return nil, nil, usageErrorf(spec.Name, "flag --%s does not take a value (got %q)", f.Name, value)
			}
			flags[f.Name] = ""
			lastBool = f.Name
			continue
		}
		if !hasValue {
			if i+1 >= len(args) || (len(args[i+1]) > 1 && args[i+1][0] == '-') {
				return nil, nil, usageErrorf(spec.Name, "flag --%s requires a value", f.Name)
			}
			i++
			value = args[i]
		}
		if value == "" {
			return nil, nil, usageErrorf(spec.Name, "flag --%s requires a value", f.Name)
		}
		flags[f.Name] = value
	}

	if len(pos) > spec.MaxPos {
		// A positional right after a boolean flag is the likely culprit,
		// wherever it sits ("use --global yes work").
		for i := range pos {
			if flag, ok := afterBool[i]; ok {
				return nil, nil, usageErrorf(spec.Name, "flag --%s does not take a value (got %q)", flag, pos[i])
			}
		}
		return nil, nil, usageErrorf(spec.Name, "unexpected argument %q", pos[spec.MaxPos])
	}
	if len(pos) < spec.MinPos {
		names := spec.positionalNames()
		missing := "arguments"
		if len(pos) < len(names) {
			missing = names[len(pos)]
		}
		return nil, nil, usageErrorf(spec.Name, "missing required argument %s", missing)
	}
	return pos, flags, nil
}

func (s cmdSpec) flag(name string) (flagSpec, bool) {
	for _, f := range s.Flags {
		if f.Name == name {
			return f, true
		}
	}
	return flagSpec{}, false
}

func (s cmdSpec) acceptedFlags() string {
	if len(s.Flags) == 0 {
		return "; it takes no flags"
	}
	names := make([]string, 0, len(s.Flags))
	for _, f := range s.Flags {
		names = append(names, "--"+f.Name)
	}
	return "; accepted flags: " + strings.Join(names, ", ")
}

// positionalNames extracts the required <placeholders> from the first usage
// line, skipping flags and their values.
func (s cmdSpec) positionalNames() []string {
	if len(s.Usage) == 0 {
		return nil
	}
	fields := strings.Fields(s.Usage[0])
	var names []string
	for i := 2; i < len(fields); i++ {
		switch tok := fields[i]; {
		case strings.HasPrefix(tok, "--"):
			i++ // skip the flag's value placeholder
		case strings.HasPrefix(tok, "<"):
			names = append(names, tok)
		}
	}
	return names
}
