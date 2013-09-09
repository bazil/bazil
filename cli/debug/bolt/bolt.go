package bolt

import (
	"flag"
	"log"
	"path/filepath"

	"bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"github.com/boltdb/bolt"
)

type boltCommand struct {
	subcommands.Description
	subcommands.Synopsis
	flag.FlagSet
	Config struct {
		Path string
	}
	State struct {
		DB *bolt.DB
	}
}

var _ = cli.Service(&boltCommand{})

func (c *boltCommand) Setup() (ok bool) {
	var err error
	path := filepath.Join(cli.Bazil.Config.DataDir.String(), c.Config.Path)
	c.State.DB, err = bolt.Open(path, 0600)
	if err != nil {
		log.Printf("cannot open database: %v", err)
		return false
	}
	return true
}

func (c *boltCommand) Teardown() (ok bool) {
	c.State.DB.Close()
	return true
}

// Bolt manages a connection to the Bolt, and is exported so
// subcommands below it can use the store.
var Bolt = boltCommand{
	Description: "Bolt key-value manipulation",
	Synopsis:    "[--path=PATH] COMMAND..",
}

func init() {
	// paths are relative to datadir
	Bolt.StringVar(&Bolt.Config.Path, "path", "bazil.bolt", "path to Bolt database, relative to datadir")

	subcommands.Register(&Bolt)
}
