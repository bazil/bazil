package list

import (
	"fmt"

	clibolt "bazil.org/bazil/cli/debug/bolt"
	"bazil.org/bazil/cliutil/subcommands"
	"github.com/boltdb/bolt"
)

type listCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		Bucket string
	}
	// TODO could support ranges and prefixes
}

func (c *listCommand) Run() error {
	buckets, err := clibolt.SplitBuckets(c.Arguments.Bucket)
	if err != nil {
		return err
	}
	err = clibolt.Bolt.State.DB.View(func(tx *bolt.Tx) error {
		bucket, err := clibolt.LookupBucket(tx, buckets)
		if err != nil {
			return err
		}
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if v == nil {
				// skip buckets
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

var list = listCommand{
	Description: "list keys in the database",
	Synopsis:    "BUCKET[{/BUCKET}..]",
}

func init() {
	subcommands.Register(&list)
}
