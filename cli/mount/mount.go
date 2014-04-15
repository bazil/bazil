package mount

import (
	"errors"
	"flag"
	"fmt"
	"path/filepath"

	"bazil.org/bazil/cas/chunks/kvchunks"
	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/fs"
	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/server"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"github.com/boltdb/bolt"
	"github.com/tv42/jog"
)

type mountCommand struct {
	subcommands.Description
	flag.FlagSet
	Arguments struct {
		Mountpoint string
	}
}

func (c *mountCommand) Run() error {
	dataDir := clibazil.Bazil.Config.DataDir.String()
	app, err := server.New(dataDir)
	if err != nil {
		return fmt.Errorf("app: %v", err)
	}
	defer app.Close()

	kvpath := filepath.Join(app.DataDir, "chunks")
	kvstore, err := kvfiles.Open(kvpath)
	if err != nil {
		return err
	}
	chunkStore := kvchunks.New(kvstore)

	// TODO hardcoded volume "default"
	var volID fs.VolumeID
	if err := app.DB.View(func(tx *bolt.Tx) error {
		val := tx.Bucket([]byte(tokens.BucketVolName)).Get([]byte("default"))
		if val == nil {
			return errors.New("volume not found")
		}
		copy(volID[:], val)
		return nil
	}); err != nil {
		return fmt.Errorf("cannot create default volume: %v", err)
	}

	filesys, err := fs.Open(app.DB, chunkStore, &volID)
	if err != nil {
		return fmt.Errorf("fs open: %v", err)
	}

	if clibazil.Bazil.Config.Debug {
		log := jog.New(nil)
		fuse.Debug = log.Event
	}

	conn, err := fuse.Mount(c.Arguments.Mountpoint)
	if err != nil {
		return fmt.Errorf("mount fail: %v", err)
	}
	defer conn.Close()

	err = fusefs.Serve(conn, filesys)
	if err != nil {
		return fmt.Errorf("filesystem failure: %v", err)
	}

	// check if the mount process has an error to report
	<-conn.Ready
	if err := conn.MountError; err != nil {
		return fmt.Errorf("mount fail (delayed): %v", err)
	}

	return nil
}

var mount = mountCommand{
	Description: "mount the filesystem",
}

func init() {
	subcommands.Register(&mount)
}
