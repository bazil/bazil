package db

import (
	"bytes"
	"encoding/binary"

	wirefs "bazil.org/bazil/fs/wire"
	"bazil.org/fuse"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
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

func basename(dirKey []byte) []byte {
	return dirKey[8:]
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

// TombstoneCreate marks the entry in parent directory with the given
// name as removed, but leaves a hint of its existence. It creates a
// new entry if one does not exist yet.
func (b *Dirs) TombstoneCreate(parentInode uint64, name string) error {
	de := &wirefs.Dirent{Tombstone: &wirefs.Tombstone{}}
	if err := b.Put(parentInode, name, de); err != nil {
		return err
	}
	return nil
}

// Rename renames an entry in the parent directory from oldName to
// newName.
//
// Returns the overwritten entry, or nil.
func (b *Dirs) Rename(parentInode uint64, oldName string, newName string) (*DirEntry, error) {
	keyOld := dirKey(parentInode, oldName)
	keyNew := dirKey(parentInode, newName)

	bufOld := b.b.Get(keyOld)
	if bufOld == nil {
		return nil, fuse.ENOENT
	}

	// the file getting overwritten
	var loser *DirEntry
	if buf := b.b.Get(keyNew); buf != nil {
		// overwriting
		loser = &DirEntry{
			name: basename(keyNew),
			data: buf,
		}
	}

	if err := b.b.Put(keyNew, bufOld); err != nil {
		return nil, err
	}
	tombDE := &wirefs.Dirent{Tombstone: &wirefs.Tombstone{}}
	tombBuf, err := proto.Marshal(tombDE)
	if err != nil {
		return nil, err
	}
	if err := b.b.Put(keyOld, tombBuf); err != nil {
		return nil, err
	}

	return loser, nil
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

// Seek to first name equal to name, or the next one if exact match is
// not found.
//
// Passing an empty name seeks to the beginning of the directory.
func (c *DirsCursor) Seek(name string) *DirEntry {
	k := make([]byte, 0, len(c.prefix)+len(name))
	k = append(k, c.prefix...)
	k = append(k, name...)
	return c.item(c.c.Seek(k))
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
