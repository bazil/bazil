package cas

import (
	"flag"
	"log"
	"path/filepath"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/kv/kvfiles"
)

type casCommand struct {
	subcommands.Description
	subcommands.Synopsis
	flag.FlagSet
	Config struct {
		Path string
	}
	State struct {
		Store chunks.Store
	}
}

var _ = cli.Service(&casCommand{})

func (c *casCommand) Setup() (ok bool) {
	path := filepath.Join(cli.Bazil.Config.DataDir.String(), c.Config.Path)
	kvstore, err := kvfiles.Open(path)
	if err != nil {
		log.Printf("cannot open CAS: %v", err)
		return false
	}

	c.State.Store = kvchunks.New(kvstore)
	return true
}

func (c *casCommand) Teardown() (ok bool) {
	return true
}

// CAS manages a connection to the Content Addressed Store, and is
// exported so subcommands below it can use the store.
var CAS = casCommand{
	Description: "low-level Content Addressed Store access",
	Synopsis:    "[--path=PATH] COMMAND..",
}

func init() {
	// paths are relative to datadir
	CAS.StringVar(&CAS.Config.Path, "path", "chunks", "path to chunk store, relative to datadir")

	subcommands.Register(&CAS)
}
