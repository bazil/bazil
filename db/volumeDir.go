package db

import (
	"encoding/binary"

	wirefs "bazil.org/bazil/fs/wire"
	"bazil.org/fuse"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
)

type Dirs struct {
	b *bolt.Bucket
}

func dirKey(parentInode uint64, name string) []byte {
	buf := make([]byte, 8+len(name))
	binary.BigEndian.PutUint64(buf, parentInode)
	copy(buf[8:], name)
	return buf
}

// Get the entry in parent directory with the given name.
//
// Returned value is valid after the transaction.
func (b *Dirs) Get(parentInode uint64, name string) (*wirefs.Dirent, error) {
	key := dirKey(parentInode, name)
	buf := b.b.Get(key)
	if buf == nil {
		return nil, fuse.ENOENT
	}

	var de wirefs.Dirent
	if err := proto.Unmarshal(buf, &de); err != nil {
		return nil, err
	}
	return &de, nil
}
