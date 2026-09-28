package cli

import (
	"fmt"
	"strings"
)

// UsageError is returned for every command-line mistake: unknown command or
// flag, missing or surplus arguments, or an invalid value given on the
// command line. cmd/ghs maps it to exit code 2.
type UsageError struct {
	Command string
	Message string
	// Usage is the command's usage block, printed after the message.
	Usage string
}

func (e *UsageError) Error() string {
	return e.Message
}

// usageErrorf builds a UsageError for a registered command.
func usageErrorf(command string, format string, a ...any) *UsageError {
	return &UsageError{Command: command, Message: fmt.Sprintf(format, a...), Usage: usageFor(command)}
}

// usageFor returns the usage block of a registered command.
func usageFor(command string) string {
	spec, ok := lookupCommand(command)
	if !ok {
		return generalUsage()
	}
	return spec.usageText()
}

// usageText is the command's lines from the command list plus one line per
// flag.
func (s cmdSpec) usageText() string {
	var b strings.Builder
	for _, line := range s.Usage {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	if len(s.Flags) == 0 {
		fmt.Fprintf(&b, "  (ghs %s takes no flags)\n", s.Name)
		return b.String()
	}
	for _, f := range s.Flags {
		name := "--" + f.Name
		if f.TakesValue {
			name += " " + f.Value
		}
		fmt.Fprintf(&b, "  %-22s %s\n", name, f.Help)
	}
	return b.String()
}

func generalUsage() string {
	var b strings.Builder
	b.WriteString("Usage:\n")
	for _, line := range commandListLines() {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\nRun \"ghs <command> --help\" for a command's flags.\n")
	return b.String()
}
