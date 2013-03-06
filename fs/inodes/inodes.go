// Package inodes contains the logic for allocating inode numbers.
//
// We just store each inode as a key in the database. The overhead is
// roughly 8 bytes per inode. Inodes are never freed, currently. 2**64
// files oughta be enough for a while.
package inodes

import (
	"encoding/binary"
	"math"
	"syscall"

	"bazil.org/fuse"
	"github.com/boltdb/bolt"
)

const maxReservedInode = 1023

func inodeToBytes(inode uint64, buf []byte) {
	binary.BigEndian.PutUint64(buf, inode)
}

func bytesToInode(buf []byte) uint64 {
	return binary.BigEndian.Uint64(buf)
}

type outOfInodesError struct{}

var _ = error(outOfInodesError{})
var _ = fuse.ErrorNumber(outOfInodesError{})

func (outOfInodesError) Error() string {
	return "out of inodes"
}

func (outOfInodesError) Errno() fuse.Errno {
	return fuse.Errno(syscall.ENOSPC)
}

var OutOfInodes = outOfInodesError{}

// Allocate returns the next available inode number, and marks it
// used.
//
// Returns OutOfInodes if there are no free inodes.
func Allocate(bucket *bolt.Bucket) (uint64, error) {
	c := bucket.Cursor()
	var i uint64
	k, _ := c.Last()
	if k != nil {
		i = bytesToInode(k)
	}

	// reserve a few inodes for internal use; currently just
	// inode 1 is the root dir
	if i < maxReservedInode {
		i = maxReservedInode
	}

	if i == math.MaxUint64 {
		return 0, OutOfInodes
	}

	i++

	var buf [8]byte
	inodeToBytes(i, buf[:])
	err := bucket.Put(buf[:], nil)
	if err != nil {
		return 0, err
	}
	return i, nil
}
