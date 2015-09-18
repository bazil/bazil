package cas

import (
	"flag"
	"log"
	"path/filepath"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/db"
	"bazil.org/bazil/kv"
	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/kv/untrusted"
	"bazil.org/bazil/server"
)

type casCommand struct {
	subcommands.Description
	subcommands.Synopsis
	flag.FlagSet
	Config struct {
		Path    string
		Sharing string
	}
	State struct {
		Store chunks.Store
	}
}

var _ cli.Service = (*casCommand)(nil)

func (c *casCommand) getSharingKey(name string, out *[32]byte) error {
	app, err := server.New(cli.Bazil.Config.DataDir.String())
	if err != nil {
		return err
	}
	defer app.Close()

	view := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Get(name)
		if err != nil {
			return err
		}
		sharingKey.Secret(out)
		return nil
	}
	if err := app.DB.View(view); err != nil {
		return err
	}
	return nil
}

func (c *casCommand) Setup() (ok bool) {
	path := filepath.Join(cli.Bazil.Config.DataDir.String(), c.Config.Path)
	var kvstore kv.KV
	var err error
	kvstore, err = kvfiles.Open(path)
	if err != nil {
		log.Printf("cannot open CAS: %v", err)
		return false
	}

	if c.Config.Sharing != "" {
		var secret [32]byte
		if err := c.getSharingKey(c.Config.Sharing, &secret); err != nil {
			log.Printf("cannot get sharing key: %q: %v", c.Config.Sharing, err)
			return false
		}
		kvstore = untrusted.New(kvstore, &secret)
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
	CAS.StringVar(&CAS.Config.Sharing, "sharing", "default", "sharing group content is encrypted for")

	subcommands.Register(&CAS)
}
