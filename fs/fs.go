package fs

import (
	"encoding/binary"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/boltdb/bolt"
)

type Volume struct {
	db         *bolt.DB
	volID      VolumeID
	chunkStore chunks.Store
}

var _ = fs.FS(&Volume{})

var bucketVolume = []byte(tokens.BucketVolume)
var bucketDir = []byte("dir")
var bucketInode = []byte("inode")

func (v *Volume) bucket(tx *bolt.Tx) *bolt.Bucket {
	b := tx.Bucket(bucketVolume)
	b = b.Bucket(v.volID.Bytes())
	return b
}

// Open returns a FUSE filesystem instance serving content from the
// given database and chunk store. The result can be passed to
// bazil.org/fuse/fs#Serve to start serving file access requests from
// the kernel.
func Open(db *bolt.DB, chunkStore chunks.Store) (*Volume, error) {
	fs := &Volume{}
	copy(fs.volID[:], "defaultvol") // TODO
	fs.db = db
	fs.chunkStore = chunkStore
	return fs, nil
}

// TODO this needs to go away
func Init(tx *bolt.Tx) error {
	b := tx.Bucket(bucketVolume)
	var volID VolumeID
	copy(volID[:], "defaultvol") // TODO
	var err error
	b, err = b.CreateBucketIfNotExists(volID.Bytes())
	if err != nil {
		return err
	}
	if _, err := b.CreateBucketIfNotExists(bucketDir); err != nil {
		return err
	}
	if _, err := b.CreateBucketIfNotExists(bucketInode); err != nil {
		return err
	}
	return nil
}

func (v *Volume) Root() (fs.Node, fuse.Error) {
	d := &dir{
		inode:  1,
		parent: nil,
		fs:     v,
		active: make(map[string]node),
	}
	return d, nil
}

func pathToKey(parentInode uint64, name string) []byte {
	buf := make([]byte, 8+len(name))
	binary.BigEndian.PutUint64(buf, parentInode)
	copy(buf[8:], name)
	return buf
}

type node interface {
	fs.Node

	marshal() (*wire.Dirent, error)
	getName() string
	setName(name string)
}
