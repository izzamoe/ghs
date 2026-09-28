package sshops

import (
	"path/filepath"
)

// ReadConfig reads home/.ssh/config without writing anything; a missing
// file is empty content.
func ReadConfig(home string) (content string, path string, err error) {
	path = filepath.Join(home, ".ssh", "config")
	content, err = ReadConfigFile(path)
	return content, path, err
}

// HasHostBlockInConfig reports whether ~/.ssh/config (under home) has a
// Host line listing alias.
func HasHostBlockInConfig(home string, alias string) (bool, error) {
	content, _, err := ReadConfig(home)
	if err != nil {
		return false, err
	}
	return hasHostBlock(content, alias), nil
}
