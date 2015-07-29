package db

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"bazil.org/bazil/fs/clock"
	wirepeer "bazil.org/bazil/peer/wire"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

// VolumeConflicts tracks the alternate versions of directory
// entries.
type VolumeConflicts struct {
	b *bolt.Bucket
}

func (VolumeConflicts) pathToKey(parentInode uint64, name string, clock []byte) []byte {
	buf := make([]byte, 8, 8+len(name)+1+len(clock))
	binary.BigEndian.PutUint64(buf, parentInode)
	buf = append(buf, name...)
	buf = append(buf, '\x00')
	buf = append(buf, clock...)
	return buf
}

func (vc *VolumeConflicts) Add(parentInode uint64, c *clock.Clock, de *wirepeer.Dirent) error {
	clockBuf, err := c.MarshalBinary()
	if err != nil {
		return fmt.Errorf("error marshaling clock: %v", err)
	}
	key := vc.pathToKey(parentInode, de.Name, clockBuf)

	tmp := *de
	tmp.Name = ""
	tmp.Clock = nil
	buf, err := proto.Marshal(&tmp)
	if err != nil {
		return fmt.Errorf("error marshaling dirent: %v", err)
	}

	if err := vc.b.Put(key, buf); err != nil {
		return err
	}
	return nil
}

func (vc *VolumeConflicts) List(parentInode uint64, name string) *VolumeConflictsCursor {
	c := vc.b.Cursor()
	prefix := vc.pathToKey(parentInode, name, nil)
	return &VolumeConflictsCursor{
		prefix: prefix,
		c:      c,
	}
}

type VolumeConflictsCursor struct {
	prefix []byte
	c      *bolt.Cursor
}

func (c *VolumeConflictsCursor) First() *VolumeConflictsItem {
	return c.item(c.c.Seek(c.prefix))
}

func (c *VolumeConflictsCursor) Next() *VolumeConflictsItem {
	return c.item(c.c.Next())
}

// Delete the current item.
func (c *VolumeConflictsCursor) Delete() error {
	return c.c.Delete()
}

func (c *VolumeConflictsCursor) item(k, v []byte) *VolumeConflictsItem {
	if !bytes.HasPrefix(k, c.prefix) {
		// past the end of the dirent
		return nil
	}
	return &VolumeConflictsItem{
		prefixLen: len(c.prefix),
		key:       k,
		data:      v,
	}
}

type VolumeConflictsItem struct {
	prefixLen int
	key       []byte
	data      []byte
}

// Clock returns the clock for this item.
//
// Returned value is valid after the transaction.
func (item *VolumeConflictsItem) Clock() (*clock.Clock, error) {
	var c clock.Clock
	clockBuf := item.key[item.prefixLen:]
	if err := c.UnmarshalBinary(clockBuf); err != nil {
		return nil, fmt.Errorf("error unmarshaling clock: %v", err)
	}
	return &c, nil
}

// Dirent returns the directory entry for this item.
//
// out is valid after the transaction.
func (item *VolumeConflictsItem) Dirent(out *wirepeer.Dirent) error {
	if err := proto.Unmarshal(item.data, out); err != nil {
		return fmt.Errorf("error unmarshaling dirent: %v", err)
	}
	return nil
}
