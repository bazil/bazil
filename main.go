package main

import (
	"os"

	"bazil.org/bazil/cli"
)

import (
	// CLI subcommands
	_ "bazil.org/bazil/cli/create"
	_ "bazil.org/bazil/cli/server/ping"
	_ "bazil.org/bazil/cli/server/run"
	_ "bazil.org/bazil/cli/version"
	_ "bazil.org/bazil/cli/volume/create"
	_ "bazil.org/bazil/cli/volume/mount"

	// CLI debug tools
	_ "bazil.org/bazil/cli/debug/bolt"
	_ "bazil.org/bazil/cli/debug/bolt/buckets"
	_ "bazil.org/bazil/cli/debug/bolt/get"
	_ "bazil.org/bazil/cli/debug/bolt/list"
	_ "bazil.org/bazil/cli/debug/bolt/put"
	_ "bazil.org/bazil/cli/debug/cas"
	_ "bazil.org/bazil/cli/debug/cas/chunk/add"
	_ "bazil.org/bazil/cli/debug/cas/chunk/get"
	_ "bazil.org/bazil/cli/debug/hash"
)

func main() {
	code := cli.Main()
	os.Exit(code)
}
