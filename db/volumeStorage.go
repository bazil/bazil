package db

import (
	"errors"

	"bazil.org/bazil/db/wire"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

var (
	ErrVolumeStorageExist = errors.New("volume storage name exists already")
)

type VolumeStorage struct {
	b *bolt.Bucket
}

// Add a storage backend to be used by the volume.
//
// Active Volume instances are not notified.
//
// If volume has storage by that name already, returns
// ErrVolumeStorageExist.
func (vs *VolumeStorage) Add(name string, backend string, sharingKey *SharingKey) error {
	n := []byte(name)
	if v := vs.b.Get(n); v != nil {
		return ErrVolumeStorageExist
	}
	msg := &wire.VolumeStorage{
		Backend:        backend,
		SharingKeyName: sharingKey.Name(),
	}
	buf, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return vs.b.Put(n, buf)
}

func (vs *VolumeStorage) Cursor() *VolumeStorageCursor {
	return &VolumeStorageCursor{vs.b.Cursor()}
}

type VolumeStorageCursor struct {
	c *bolt.Cursor
}

func (c *VolumeStorageCursor) item(k, v []byte) *VolumeStorageItem {
	if k == nil {
		return nil
	}
	return &VolumeStorageItem{name: k, data: v}
}

func (c *VolumeStorageCursor) First() *VolumeStorageItem {
	return c.item(c.c.First())
}

func (c *VolumeStorageCursor) Next() *VolumeStorageItem {
	return c.item(c.c.Next())
}

type VolumeStorageItem struct {
	name []byte
	data []byte
	conf wire.VolumeStorage
}

func (item *VolumeStorageItem) unmarshal() error {
	return proto.Unmarshal(item.data, &item.conf)
}

// Backend returns the storage backend for this item.
//
// Returned value is valid after the transaction.
func (item *VolumeStorageItem) Backend() (string, error) {
	if item.conf.Backend == "" {
		if err := item.unmarshal(); err != nil {
			return "", err
		}
	}
	return item.conf.Backend, nil
}

// SharingKeyName returns the sharing key name for this item.
//
// Returned value is valid after the transaction.
func (item *VolumeStorageItem) SharingKeyName() (string, error) {
	if item.conf.Backend == "" {
		if err := item.unmarshal(); err != nil {
			return "", err
		}
	}
	return item.conf.SharingKeyName, nil
}
