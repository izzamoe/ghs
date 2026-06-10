package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func DefaultPath() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome != "" {
		return filepath.Join(configHome, "ghs", "config.conf"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmtHomeError(err)
	}
	return filepath.Join(home, ".config", "ghs", "config.conf"), nil
}

func ExpandPath(path string) (string, error) {
	if path == "" || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmtHomeError(err)
	}
	if path == "~" {
		return home, nil
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}
	return "", errors.New("unsupported home path format")
}

func fmtHomeError(err error) error {
	return fmt.Errorf("cannot resolve user home directory: %w", err)
}
