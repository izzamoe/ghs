package cli

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"runtime/debug"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
)

type App struct {
	out io.Writer
	err io.Writer
}

func New(out io.Writer, err io.Writer) App {
	return App{out: out, err: err}
}

// commands is the single source of truth for the CLI grammar: help text,
// README (enforced by docs_test.go), usage errors, and parsing all derive
// from it. The Usage lines match specs/001-ghs-context-safety/contracts/cli.md.
var commands = []cmdSpec{
	{
		Name:   "add-profile",
		Usage:  []string{"ghs add-profile <name> --gh-user <login> --git-name <name> --git-email <email> --ssh-alias <alias> --ssh-key <path> [--workspace <path>]"},
		MinPos: 1, MaxPos: 1,
		Flags: []flagSpec{
			{Name: "gh-user", TakesValue: true, Value: "<login>", Help: "GitHub login of the account (required)"},
			{Name: "git-name", TakesValue: true, Value: "<name>", Help: "Git author name (required)"},
			{Name: "git-email", TakesValue: true, Value: "<email>", Help: "Git author email (required)"},
			{Name: "ssh-alias", TakesValue: true, Value: "<alias>", Help: "SSH host alias, e.g. github-work (required)"},
			{Name: "ssh-key", TakesValue: true, Value: "<path>", Help: "SSH private key path, absolute or ~/ (required)"},
			{Name: "workspace", TakesValue: true, Value: "<path>", Help: "link this directory to the profile's Git identity"},
		},
	},
	{
		Name:   "add-from-gh",
		Usage:  []string{"ghs add-from-gh <name> [--git-name <name>] [--git-email <email>] [--ssh-alias <alias>] [--ssh-key <path>] [--workspace <path>] [--require-email]"},
		MinPos: 1, MaxPos: 1,
		Flags: []flagSpec{
			{Name: "git-name", TakesValue: true, Value: "<name>", Help: "Git author name (default: the GitHub name)"},
			{Name: "git-email", TakesValue: true, Value: "<email>", Help: "Git author email (default: the GitHub email or noreply address)"},
			{Name: "ssh-alias", TakesValue: true, Value: "<alias>", Help: "SSH host alias (default: github-<name>)"},
			{Name: "ssh-key", TakesValue: true, Value: "<path>", Help: "SSH private key path (default: ~/.ssh/id_ed25519_<name>)"},
			{Name: "workspace", TakesValue: true, Value: "<path>", Help: "link this directory to the profile's Git identity"},
			{Name: "require-email", Help: "fail instead of saving a profile without an email"},
		},
	},
	{
		Name:  "import-all",
		Usage: []string{"ghs import-all [--hostname github.com] [--require-email] [--no-overwrite]"},
		Flags: []flagSpec{
			{Name: "hostname", TakesValue: true, Value: "github.com", Help: "GitHub CLI host; only github.com is supported"},
			{Name: "require-email", Help: "fail when an account has no email"},
			{Name: "no-overwrite", Help: "keep existing profiles with the same name"},
		},
	},
	{Name: "list", Usage: []string{"ghs list"}},
	{Name: "status", Usage: []string{"ghs status"}},
	{
		Name:   "doctor",
		Usage:  []string{"ghs doctor [<profile>] [--offline]"},
		MaxPos: 1,
		Flags:  []flagSpec{{Name: "offline", Help: "skip the ssh -T authentication check (no network)"}},
	},
	{Name: "set-email", Usage: []string{"ghs set-email <profile> <email>"}, MinPos: 2, MaxPos: 2},
	{
		Name:   "workspace",
		Usage:  []string{"ghs workspace <profile> <path>", "ghs workspace <profile> --unlink"},
		MinPos: 1, MaxPos: 2,
		Flags: []flagSpec{{Name: "unlink", Help: "remove the profile's workspace link"}},
	},
	{
		Name:   "use",
		Usage:  []string{"ghs use <profile> [--global] [--fix-remote]"},
		MinPos: 1, MaxPos: 1,
		Flags: []flagSpec{
			{Name: "global", Help: "set the global Git identity instead of the repository's"},
			{Name: "fix-remote", Help: "first rewrite origin to go through the profile's SSH alias"},
		},
	},
	{
		Name:   "clone",
		Usage:  []string{"ghs clone <profile> <owner/repo|github-url> [directory] [--upload-key]"},
		MinPos: 2, MaxPos: 3,
		Flags: []flagSpec{{Name: "upload-key", Help: "upload the SSH public key to the profile's account"}},
	},
	{
		Name:   "init-ssh",
		Usage:  []string{"ghs init-ssh <profile> [--upload]"},
		MinPos: 1, MaxPos: 1,
		Flags: []flagSpec{{Name: "upload", Help: "upload the SSH public key to the profile's account"}},
	},
	{Name: "fix-remote", Usage: []string{"ghs fix-remote <profile>"}, MinPos: 1, MaxPos: 1},
	{Name: "remove", Usage: []string{"ghs remove <profile>"}, MinPos: 1, MaxPos: 1},
	{Name: "version", Usage: []string{"ghs version"}},
	{Name: "update", Usage: []string{"ghs update"}},
}

func lookupCommand(name string) (cmdSpec, bool) {
	for _, spec := range commands {
		if spec.Name == name {
			return spec, true
		}
	}
	return cmdSpec{}, false
}

// commandListLines is the command list printed by help and copied verbatim
// into README.md.
func commandListLines() []string {
	var lines []string
	for _, spec := range commands {
		lines = append(lines, spec.Usage...)
	}
	return lines
}

// reservedNames are words a profile name may not be.
func reservedNames() []string {
	names := []string{"help"}
	for _, spec := range commands {
		names = append(names, spec.Name)
	}
	return names
}

func (a App) PrintError(err error) {
	var usageErr *UsageError
	if errors.As(err, &usageErr) {
		_, _ = fmt.Fprintln(a.err, "ghs:", usageErr.Message)
		if usageErr.Usage != "" {
			_, _ = fmt.Fprint(a.err, "\n"+usageErr.Usage)
		}
		return
	}
	for line := range strings.SplitSeq(err.Error(), "\n") {
		if line != "" {
			_, _ = fmt.Fprintln(a.err, "ghs:", line)
		}
	}
}

// warnf prints a warning to stderr; warnings never change the exit code.
func (a App) warnf(format string, args ...any) {
	_, _ = fmt.Fprintf(a.err, "ghs: warning: "+format+"\n", args...)
}

// printf writes one line of success output.
func (a App) printf(format string, args ...any) error {
	_, err := fmt.Fprintf(a.out, format+"\n", args...)
	return err
}

func (a App) Run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return a.printHelp()
	}
	name := args[0]
	if name == "--version" || name == "-v" {
		name = "version"
	}
	spec, ok := lookupCommand(name)
	if !ok {
		return &UsageError{Message: fmt.Sprintf("unknown command %q", name), Usage: generalUsage()}
	}
	pos, flags, err := parseArgs(spec, args[1:])
	if errors.Is(err, errHelp) {
		_, err := fmt.Fprint(a.out, spec.usageText())
		return err
	}
	if err != nil {
		return err
	}

	switch spec.Name {
	case "add-profile":
		return a.addProfile(pos, flags)
	case "add-from-gh":
		return a.addFromGH(pos, flags)
	case "import-all":
		return a.importAll(flags)
	case "list":
		return a.listProfiles()
	case "status":
		return a.status()
	case "set-email":
		return a.setEmail(pos)
	case "use":
		return a.useProfile(pos, flags)
	case "clone":
		return a.clone(pos, flags)
	case "init-ssh":
		return a.initSSH(pos, flags)
	case "fix-remote":
		return a.fixRemote(pos)
	case "workspace":
		return a.workspace(pos, flags)
	case "remove":
		return a.remove(pos)
	case "version":
		return a.printVersion()
	case "update":
		return a.update()
	}
	return fmt.Errorf("command %q is not implemented", spec.Name)
}

func (a App) printHelp() error {
	var b strings.Builder
	b.WriteString("ghs - GitHub account switch helper\n\n")
	b.WriteString(generalUsage())
	b.WriteString(`
Config: $XDG_CONFIG_HOME/ghs/config.conf, or ~/.config/ghs/config.conf
Only github.com is supported.
Do not run ghs with sudo.
Exit codes: 0 success, 1 failure, 2 usage error.
`)
	_, err := fmt.Fprint(a.out, b.String())
	return err
}

func (a App) printVersion() error {
	_, err := fmt.Fprintln(a.out, "ghs", buildVersion())
	return err
}

func buildVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "devel"
}

// loadConfig loads the config strictly: a missing file is an empty config;
// an unreadable or malformed file is fatal and is never treated as empty.
func (a App) loadConfig() (string, config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return "", config.Config{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return path, config.Config{}, nil
		}
		return "", config.Config{}, err
	}
	return path, cfg, nil
}

// findProfile looks a profile up exactly, suggesting a case variant on a
// miss.
func findProfile(cfg config.Config, name string) (config.Profile, int, error) {
	if i := cfg.Index(name); i >= 0 {
		return cfg.Profiles[i], i, nil
	}
	if other, ok := cfg.FindProfileFold(name); ok {
		return config.Profile{}, -1, fmt.Errorf("profile %q not found; did you mean %q?", name, other.Name)
	}
	return config.Profile{}, -1, fmt.Errorf("profile %q not found", name)
}

func (a App) loadProfile(name string) (config.Profile, error) {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return config.Profile{}, err
	}
	profile, _, err := findProfile(cfg, name)
	return profile, err
}
