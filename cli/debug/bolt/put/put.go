package put

import (
	"io/ioutil"
	"os"

	"github.com/boltdb/bolt"

	clibolt "bazil.org/bazil/cli/debug/bolt"
	"bazil.org/bazil/cliutil/subcommands"
)

type putCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		Bucket string
		Key    string
	}
}

func (c *putCommand) Run() error {
	buckets, err := clibolt.SplitBuckets(c.Arguments.Bucket)
	if err != nil {
		return err
	}

	key, err := clibolt.DecodeKey(c.Arguments.Key)
	if err != nil {
		return err
	}

	val, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	err = clibolt.Bolt.State.DB.Update(func(tx *bolt.Tx) error {
		bucket, err := clibolt.LookupBucket(tx, buckets)
		if err != nil {
			return err
		}
		return bucket.Put(key, val)
	})
	if err != nil {
		return err
	}
	return nil
}

var put = putCommand{
	Description: "put a value into the database",
	Synopsis:    "BUCKET[{/BUCKET}..] KEY <FILE",
}

func init() {
	subcommands.Register(&put)
}
