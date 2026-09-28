package sshops

import (
	"path/filepath"
	"strings"

	"github.com/izzamoe/ghs/internal/config"
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

// HostBlock holds the keywords doctor checks from one Host block. Values
// are unquoted; IdentityFile has a leading ~ expanded.
type HostBlock struct {
	Found          bool
	HostName       string
	User           string
	IdentityFile   string
	IdentitiesOnly string
}

// ParseHostBlock finds the first Host block listing alias (keywords and
// patterns match case-insensitively; "=" or whitespace separates keyword
// and value) and collects its keywords until the next Host or Match line.
// Within the block the first value of each keyword wins, as in ssh_config.
func ParseHostBlock(content string, alias string) HostBlock {
	var block HostBlock
	for line := range strings.SplitSeq(content, "\n") {
		keyword, rest := splitKeyword(line)
		if keyword == "" {
			continue
		}
		switch {
		case strings.EqualFold(keyword, "Host"):
			if block.Found {
				return block
			}
			for word := range strings.FieldsSeq(rest) {
				if strings.EqualFold(strings.Trim(word, `"`), alias) {
					block.Found = true
				}
			}
			continue
		case strings.EqualFold(keyword, "Match"):
			if block.Found {
				return block
			}
			continue
		}
		if !block.Found {
			continue
		}
		value := unquote(rest)
		set := func(field *string) {
			if *field == "" {
				*field = value
			}
		}
		switch strings.ToLower(keyword) {
		case "hostname":
			set(&block.HostName)
		case "user":
			set(&block.User)
		case "identitiesonly":
			set(&block.IdentitiesOnly)
		case "identityfile":
			if block.IdentityFile == "" {
				if expanded, err := config.ExpandPath(value); err == nil {
					value = expanded
				}
				block.IdentityFile = value
			}
		}
	}
	return block
}

func unquote(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return v[1 : len(v)-1]
	}
	return v
}
