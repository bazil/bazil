package main

import (
	"os"

	"bazil.org/bazil/cli"
)

import (
	// CLI subcommands
	_ "bazil.org/bazil/cli/version"
)

func main() {
	code := cli.Main()
	os.Exit(code)
}
