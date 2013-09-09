package get

import (
	"errors"
	"os"

	"github.com/boltdb/bolt"

	clibolt "bazil.org/bazil/cli/debug/bolt"
	"bazil.org/bazil/cliutil/subcommands"
)

type getCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		Bucket string
		Key    string
	}
}

func (c *getCommand) Run() error {
	buckets, err := clibolt.SplitBuckets(c.Arguments.Bucket)
	if err != nil {
		return err
	}

	key, err := clibolt.DecodeKey(c.Arguments.Key)
	if err != nil {
		return err
	}

	var val []byte
	err = clibolt.Bolt.State.DB.View(func(tx *bolt.Tx) error {
		bucket, err := clibolt.LookupBucket(tx, buckets)
		if err != nil {
			return err
		}
		val = bucket.Get(key)
		return nil
	})
	if err != nil {
		return err
	}
	if val == nil {
		return errors.New("database key not found")
	}
	_, err = os.Stdout.Write(val)
	if err != nil {
		return err
	}
	return nil
}

var get = getCommand{
	Description: "get a value from the database",
	Synopsis:    "BUCKET[{/BUCKET}..] KEY >FILE",
}

func init() {
	subcommands.Register(&get)
}
