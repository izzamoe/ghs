package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
	"github.com/izzamoe/ghs/internal/gitops"
	"github.com/izzamoe/ghs/internal/runner"
)

// workspace links a directory to a profile's Git identity with a global
// includeIf entry pointing at a ghs-owned identity file, or unlinks it.
func (a App) workspace(pos []string, flags map[string]string) error {
	unlink := hasFlagKey(flags, "unlink")
	switch {
	case unlink && len(pos) == 2:
		return usageErrorf("workspace", "give either a path or --unlink, not both")
	case !unlink && len(pos) == 1:
		return usageErrorf("workspace", "missing required argument <path> (or pass --unlink)")
	}
	if !unlink {
		if err := config.ValidateWorkspace(pos[1]); err != nil {
			return usageErrorf("workspace", "%v", err)
		}
	}
	path, cfg, err := a.loadConfig()
	if err != nil {
		return err
	}
	_, idx, err := findProfile(cfg, pos[0])
	if err != nil {
		return err
	}
	if unlink {
		return a.workspaceUnlink(path, cfg, idx)
	}
	return a.workspaceLink(path, cfg, idx, pos[1])
}

// workspaceLinkParts derives the includeIf key and the identity file (native
// path and the forward-slash form stored in Git config and printed).
func workspaceLinkParts(cfgPath string, p config.Profile, workspace string) (key, file, fileSlash string) {
	key = gitops.IncludeIfKey(gitops.WorkspacePattern(workspace))
	file = config.IdentityFilePath(cfgPath, p.Name)
	return key, file, filepath.ToSlash(file)
}

// workspaceLink moves any link state to "linked", adding only the missing
// parts: identity file, includeIf entry, workspace key (FR-054, FR-057).
func (a App) workspaceLink(cfgPath string, cfg config.Config, idx int, workspace string) error {
	p := cfg.Profiles[idx]
	norm := config.NormalizeWorkspace(workspace)

	// Preflight (FR-059, FR-061).
	if p.GitEmail == "" {
		return fmt.Errorf("profile %q has no email; run: ghs set-email %s <email>", p.Name, p.Name)
	}
	if err := validateIdentityFields(p); err != nil {
		return err
	}
	if p.Workspace != "" && !strings.EqualFold(config.NormalizeWorkspace(p.Workspace), norm) {
		return fmt.Errorf("profile %q is already linked to %s; run: ghs workspace %s --unlink", p.Name, p.Workspace, p.Name)
	}
	candidate := p
	candidate.Workspace = norm
	if err := cfg.CheckUnique(candidate, idx); err != nil {
		return err
	}
	if err := requireTool("git"); err != nil {
		return err
	}
	git := gitops.New(runner.New())
	key, file, fileSlash := workspaceLinkParts(cfgPath, p, norm)
	hasEntry, err := git.HasInclude(key, fileSlash)
	if err != nil {
		return fmt.Errorf("cannot read global git config: %w", err)
	}

	// Mutations, in order, each reported once it happened.
	changed := false
	if wrote, err := config.WriteIdentityFile(file, p.GitName, p.GitEmail); err != nil {
		return err
	} else if wrote {
		changed = true
		if err := a.printf("wrote identity file %s", fileSlash); err != nil {
			return err
		}
	}
	if !hasEntry {
		if err := git.AddInclude(key, fileSlash); err != nil {
			return err
		}
		changed = true
		if err := a.printf("added git include: %s = %s", key, fileSlash); err != nil {
			return err
		}
	}
	if p.Workspace != norm {
		cfg.Profiles[idx].Workspace = norm
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		changed = true
		if err := a.printf("saved workspace %s for profile %q", norm, p.Name); err != nil {
			return err
		}
	}
	if !changed {
		if err := a.printf("workspace %s is already linked for profile %q", norm, p.Name); err != nil {
			return err
		}
	}
	if expanded, err := config.ExpandPath(norm); err == nil {
		if _, err := os.Stat(expanded); errors.Is(err, fs.ErrNotExist) {
			return a.printf("note: %s does not exist yet; the identity applies once repositories exist under it", expanded)
		}
	}
	return nil
}

// unlinkWorkspaceParts removes the includeIf entry and the identity file.
// Absent parts are not errors (FR-058). It does not touch the config.
func unlinkWorkspaceParts(cfgPath string, p config.Profile) (lines []string, err error) {
	if err := requireTool("git"); err != nil {
		return nil, err
	}
	key, file, fileSlash := workspaceLinkParts(cfgPath, p, config.NormalizeWorkspace(p.Workspace))
	removed, err := gitops.New(runner.New()).RemoveInclude(key, fileSlash)
	if err != nil {
		return nil, fmt.Errorf("remove git include %s: %w", key, err)
	}
	if removed {
		lines = append(lines, fmt.Sprintf("removed git include: %s = %s", key, fileSlash))
	}
	deleted, err := config.RemoveIdentityFile(file)
	if err != nil {
		return lines, err
	}
	if deleted {
		lines = append(lines, "deleted identity file "+fileSlash)
	}
	return lines, nil
}

func (a App) workspaceUnlink(cfgPath string, cfg config.Config, idx int) error {
	p := cfg.Profiles[idx]
	if p.Workspace == "" {
		return a.printf("profile %q has no workspace", p.Name)
	}
	lines, err := unlinkWorkspaceParts(cfgPath, p)
	for _, line := range lines {
		if perr := a.printf("%s", line); perr != nil {
			return perr
		}
	}
	if err != nil {
		return err
	}
	cfg.Profiles[idx].Workspace = ""
	if err := config.Save(cfgPath, cfg); err != nil {
		return err
	}
	return a.printf("removed workspace from profile %q", p.Name)
}

// refreshIdentityFile rewrites a linked profile's identity file when it
// exists and no longer matches the profile (FR-060). It returns the line to
// print, or "".
func refreshIdentityFile(cfgPath string, p config.Profile) (string, error) {
	if p.Workspace == "" || p.GitEmail == "" {
		return "", nil
	}
	file := config.IdentityFilePath(cfgPath, p.Name)
	if _, err := os.Stat(file); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	changed, err := config.WriteIdentityFile(file, p.GitName, p.GitEmail)
	if err != nil || !changed {
		return "", err
	}
	return "updated identity file " + filepath.ToSlash(file), nil
}
