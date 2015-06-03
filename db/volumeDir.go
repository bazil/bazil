package db

import (
	"bytes"
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

// Put an entry in parent directory with the given name.
func (b *Dirs) Put(parentInode uint64, name string, de *wirefs.Dirent) error {
	buf, err := proto.Marshal(de)
	if err != nil {
		return err
	}
	key := dirKey(parentInode, name)
	if err := b.b.Put(key, buf); err != nil {
		return err
	}
	return nil
}

// Delete the entry in parent directory with the given name.
//
// Returns fuse.ENOENT if an entry does not exist.
func (b *Dirs) Delete(parentInode uint64, name string) error {
	key := dirKey(parentInode, name)
	if b.b.Get(key) == nil {
		return fuse.ENOENT
	}
	if err := b.b.Delete(key); err != nil {
		return err
	}
	return nil
}

func (b *Dirs) List(inode uint64) *DirsCursor {
	c := b.b.Cursor()
	prefix := dirKey(inode, "")
	return &DirsCursor{
		inode:  inode,
		prefix: prefix,
		c:      c,
	}
}

type DirsCursor struct {
	inode  uint64
	prefix []byte
	c      *bolt.Cursor
}

func (c *DirsCursor) First() *DirEntry {
	return c.item(c.c.Seek(c.prefix))
}

func (c *DirsCursor) Next() *DirEntry {
	return c.item(c.c.Next())
}

func (c *DirsCursor) item(k, v []byte) *DirEntry {
	if !bytes.HasPrefix(k, c.prefix) {
		// past the end of the directory
		return nil
	}
	name := k[len(c.prefix):]
	return &DirEntry{name: name, data: v}
}

type DirEntry struct {
	name []byte
	data []byte
}

// Name returns the basename of this directory entry.
//
// name is valid after the transaction.
func (e *DirEntry) Name() string {
	return string(e.name)
}

// Unmarshal the directory entry to out.
//
// out is valid after the transaction.
func (e *DirEntry) Unmarshal(out *wirefs.Dirent) error {
	if err := proto.Unmarshal(e.data, out); err != nil {
		return err
	}
	return nil
}
