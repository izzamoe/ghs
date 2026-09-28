package main

import (
	"errors"
	"os"

	"github.com/izzamoe/ghs/internal/cli"
)

// Exit codes: 0 success, 1 any runtime failure, 2 usage error.
func main() {
	app := cli.New(os.Stdout, os.Stderr)
	if err := app.Run(os.Args[1:]); err != nil {
		app.PrintError(err)
		var usageErr *cli.UsageError
		if errors.As(err, &usageErr) {
			os.Exit(2)
		}
		os.Exit(1)
	}
}
