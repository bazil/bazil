package main

import (
	"os"

	"bazil.org/bazil/cli"
)

import (
	// CLI subcommands
	_ "bazil.org/bazil/cli/create"
	_ "bazil.org/bazil/cli/peer/add"
	_ "bazil.org/bazil/cli/peer/location/set"
	_ "bazil.org/bazil/cli/server/ping"
	_ "bazil.org/bazil/cli/server/run"
	_ "bazil.org/bazil/cli/sharing/add"
	_ "bazil.org/bazil/cli/version"
	_ "bazil.org/bazil/cli/volume/create"
	_ "bazil.org/bazil/cli/volume/mount"
	_ "bazil.org/bazil/cli/volume/storage/add"

	// CLI debug tools
	_ "bazil.org/bazil/cli/debug/cas"
	_ "bazil.org/bazil/cli/debug/cas/chunk/add"
	_ "bazil.org/bazil/cli/debug/cas/chunk/get"
	_ "bazil.org/bazil/cli/debug/hash"
	_ "bazil.org/bazil/cli/debug/peer/ping"
	_ "bazil.org/bazil/cli/debug/pubkey"
)

func main() {
	code := cli.Main()
	os.Exit(code)
}
