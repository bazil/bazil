package main

import (
	"os"

	"bazil.org/bazil/cli"
)

import (
	// CLI subcommands
	_ "bazil.org/bazil/cli/version"

	// CLI debug tools
	_ "bazil.org/bazil/cli/debug/cas"
	_ "bazil.org/bazil/cli/debug/cas/chunk/add"
	_ "bazil.org/bazil/cli/debug/cas/chunk/get"
	_ "bazil.org/bazil/cli/debug/hash"
)

func main() {
	code := cli.Main()
	os.Exit(code)
}
