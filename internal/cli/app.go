package cli

import (
	"fmt"
	"io"
	"slices"

	"github.com/izzamoe/ghs/internal/config"
)

type App struct {
	out io.Writer
	err io.Writer
}

func New(out io.Writer, err io.Writer) App {
	return App{out: out, err: err}
}

func (a App) PrintError(err error) {
	_, _ = fmt.Fprintln(a.err, "ghs:", err)
}

func (a App) Run(args []string) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		return a.printHelp()
	}

	switch args[0] {
	case "add-profile":
		return a.addProfile(args[1:])
	case "add-from-gh":
		return a.addFromGH(args[1:])
	case "import-all":
		return a.importAll(args[1:])
	case "list":
		return a.listProfiles()
	case "set-email":
		return a.setEmail(args[1:])
	case "use":
		return a.useProfile(args[1:])
	case "clone":
		return a.clone(args[1:])
	case "init-ssh":
		return a.initSSH(args[1:])
	case "fix-remote":
		return a.fixRemote(args[1:])
	case "status":
		return a.status()
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (a App) printHelp() error {
	_, err := fmt.Fprintln(a.out, `ghs - GitHub account switch helper

Commands:
  ghs add-profile <name> --gh-user <user> --git-name <name> --git-email <email> --ssh-alias <alias> --ssh-key <path> [--workspace <path>]
  ghs add-from-gh <name> [--git-name <name>] [--git-email <email>] [--ssh-alias <alias>] [--ssh-key <path>] [--workspace <path>] [--require-email]
  ghs import-all [--hostname github.com] [--require-email] [--no-overwrite]
  ghs list
  ghs set-email <profile> <email>
  ghs use <profile> [--global]
  ghs clone <profile> <owner/repo|github-url> [directory] [--upload-key]
  ghs init-ssh <profile> [--upload]
  ghs fix-remote <profile>
  ghs status

Config: ~/.config/ghs/config.conf
Root is not needed and should not be used.`)

	return err
}

func (a App) loadConfig() (string, config.Config, error) {
	path, err := config.DefaultPath()
	if err != nil {
		return "", config.Config{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return "", config.Config{}, err
	}
	return path, cfg, nil
}

func (a App) loadProfile(name string) (config.Profile, error) {
	_, cfg, err := a.loadConfig()
	if err != nil {
		return config.Profile{}, err
	}
	profile, ok := cfg.FindProfile(name)
	if !ok {
		return config.Profile{}, fmt.Errorf("profile %q not found", name)
	}
	return profile, nil
}

func hasFlag(args []string, flag string) bool {
	return slices.Contains(args, flag)
}
