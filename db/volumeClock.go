package db

import (
	"encoding/binary"
	"fmt"

	"bazil.org/bazil/fs/clock"
	"github.com/boltdb/bolt"
)

type ClockNotFoundError struct {
	ParentInode uint64
	Name        string
}

var _ error = (*ClockNotFoundError)(nil)

func (e *ClockNotFoundError) Error() string {
	return fmt.Sprintf("clock not found for %d:%q", e.ParentInode, e.Name)
}

// VolumeClock tracks the logical clocks for files and directories
// stored in the volume.
//
// Logical clocks are kept separate from the directory entries, as
// they modified at different rates. Specifically, Vector Time Pairs
// cause modification times to trickle upward in the tree, and keeping
// the clocks separate allows us to do this as a pure database
// operation, without coordinating with the active FS Node objects.
type VolumeClock struct {
	b *bolt.Bucket
}

func (VolumeClock) pathToKey(parentInode uint64, name string) []byte {
	buf := make([]byte, 8+len(name))
	binary.BigEndian.PutUint64(buf, parentInode)
	copy(buf[8:], name)
	return buf
}

func (vc *VolumeClock) Get(parentInode uint64, name string) (*clock.Clock, error) {
	key := vc.pathToKey(parentInode, name)
	val := vc.b.Get(key)
	if val == nil {
		return nil, &ClockNotFoundError{ParentInode: parentInode, Name: name}
	}
	var c clock.Clock
	if err := c.UnmarshalBinary(val); err != nil {
		return nil, err
	}
	return &c, nil
}

func (vc *VolumeClock) Put(parentInode uint64, name string, c *clock.Clock) error {
	key := vc.pathToKey(parentInode, name)
	buf, err := c.MarshalBinary()
	if err != nil {
		return err
	}
	if err := vc.b.Put(key, buf); err != nil {
		return err
	}
	return nil
}

func (vc *VolumeClock) Create(parentInode uint64, name string, now clock.Epoch) (*clock.Clock, error) {
	c := clock.Create(0, now)
	buf, err := c.MarshalBinary()
	if err != nil {
		return nil, err
	}
	key := vc.pathToKey(parentInode, name)
	if err := vc.b.Put(key, buf); err != nil {
		return nil, err
	}
	return c, nil
}

func (vc *VolumeClock) Update(parentInode uint64, name string, now clock.Epoch) (c *clock.Clock, changed bool, err error) {
	key := vc.pathToKey(parentInode, name)
	val := vc.b.Get(key)
	if val == nil {
		return nil, false, &ClockNotFoundError{ParentInode: parentInode, Name: name}
	}
	c = &clock.Clock{}
	if err := c.UnmarshalBinary(val); err != nil {
		return nil, false, err
	}
	c.Update(0, now)
	// TODO make clock.Update return changed bool
	changed = true
	if !changed {
		return nil, false, nil
	}
	buf, err := c.MarshalBinary()
	if err != nil {
		return nil, true, err
	}
	if err := vc.b.Put(key, buf); err != nil {
		return nil, true, err
	}
	return c, true, nil
}

func (vc *VolumeClock) UpdateOrCreate(parentInode uint64, name string, now clock.Epoch) (c *clock.Clock, changed bool, err error) {
	key := vc.pathToKey(parentInode, name)
	val := vc.b.Get(key)
	if val == nil {
		c = clock.Create(0, now)
		changed = true
	} else {
		c = &clock.Clock{}
		if err := c.UnmarshalBinary(val); err != nil {
			return nil, false, err
		}
		c.Update(0, now)
		// TODO make clock.UpdateOrCreate return changed bool
		changed = true
	}
	if !changed {
		return nil, false, nil
	}
	buf, err := c.MarshalBinary()
	if err != nil {
		return nil, true, err
	}
	if err := vc.b.Put(key, buf); err != nil {
		return nil, true, err
	}
	return c, true, nil
}

func (vc *VolumeClock) UpdateFromChild(parentInode uint64, name string, child *clock.Clock) (changed bool, err error) {
	key := vc.pathToKey(parentInode, name)
	val := vc.b.Get(key)
	if val == nil {
		return false, &ClockNotFoundError{ParentInode: parentInode, Name: name}
	}
	var parentClock clock.Clock
	if err := parentClock.UnmarshalBinary(val); err != nil {
		return false, fmt.Errorf("corrupt clock for %d:%q", parentInode, name)
	}
	if changed := parentClock.UpdateFromChild(child); !changed {
		// no need to persist anything
		return false, nil
	}
	buf, err := parentClock.MarshalBinary()
	if err != nil {
		return false, err
	}
	if err := vc.b.Put(key, buf); err != nil {
		return false, err
	}
	return true, nil
}

func (vc *VolumeClock) Tombstone(parentInode uint64, name string) error {
	key := vc.pathToKey(parentInode, name)
	val := vc.b.Get(key)
	if val == nil {
		return &ClockNotFoundError{ParentInode: parentInode, Name: name}
	}
	c := &clock.Clock{}
	if err := c.UnmarshalBinary(val); err != nil {
		return err
	}
	c.Tombstone()
	buf, err := c.MarshalBinary()
	if err != nil {
		return err
	}
	if err := vc.b.Put(key, buf); err != nil {
		return err
	}
	return nil
}
