package main

import (
	"os"

	"github.com/izzamoe/ghs/internal/cli"
)

func main() {
	app := cli.New(os.Stdout, os.Stderr)
	if err := app.Run(os.Args[1:]); err != nil {
		app.PrintError(err)
		os.Exit(1)
	}
}
