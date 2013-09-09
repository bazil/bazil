package buckets

import (
	"fmt"

	clibolt "bazil.org/bazil/cli/debug/bolt"
	"bazil.org/bazil/cliutil/positional"
	"bazil.org/bazil/cliutil/subcommands"
	"github.com/boltdb/bolt"
)

type bucketsCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		positional.Optional
		Bucket *string
	}
	// TODO could support ranges and prefixes
}

func (c *bucketsCommand) runRoot() error {
	err := clibolt.Bolt.State.DB.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			_, err := fmt.Println(clibolt.EncodeKey(name))
			return err
		})
	})
	return err
}

func (c *bucketsCommand) runSub(buckets [][]byte) error {
	err := clibolt.Bolt.State.DB.View(func(tx *bolt.Tx) error {
		bucket, err := clibolt.LookupBucket(tx, buckets)
		if err != nil {
			return err
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v != nil {
				// not a bucket
				continue
			}
			_, err := fmt.Println(clibolt.EncodeKey(k))
			if err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func (c *bucketsCommand) Run() error {
	if c.Arguments.Bucket == nil {
		return c.runRoot()
	} else {
		buckets, err := clibolt.SplitBuckets(*c.Arguments.Bucket)
		if err != nil {
			return err
		}

		return c.runSub(buckets)
	}
}

var buckets = bucketsCommand{
	Description: "list buckets in the database",
	Synopsis:    "BUCKET[{/BUCKET}..]",
}

func init() {
	subcommands.Register(&buckets)
}
