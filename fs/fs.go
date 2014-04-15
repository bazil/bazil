package fs

import (
	"encoding/binary"
	"errors"
	"sync"

	"crypto/rand"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/fs/inodes"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"code.google.com/p/goprotobuf/proto"
	"github.com/boltdb/bolt"
)

type Volume struct {
	mu         sync.Mutex
	db         *bolt.DB
	volID      VolumeID
	chunkStore chunks.Store
}

var _ = fs.FS(&Volume{})
var _ = fs.FSIniter(&Volume{})

var bucketVolume = []byte(tokens.BucketVolume)
var bucketVolName = []byte(tokens.BucketVolName)
var bucketDir = []byte("dir")
var bucketInode = []byte("inode")
var bucketSnap = []byte("snap")

func (v *Volume) bucket(tx *bolt.Tx) *bolt.Bucket {
	b := tx.Bucket(bucketVolume)
	b = b.Bucket(v.volID.Bytes())
	return b
}

// Open returns a FUSE filesystem instance serving content from the
// given volume. The result can be passed to bazil.org/fuse/fs#Serve
// to start serving file access requests from the kernel.
//
// Caller guarantees volume ID exists at least as long as requests are
// served for this file system.
func Open(db *bolt.DB, chunkStore chunks.Store, volumeID *VolumeID) (*Volume, error) {
	fs := &Volume{}
	fs.db = db
	fs.volID = *volumeID
	fs.chunkStore = chunkStore
	return fs, nil
}

// Create a new volume.
func Create(db *bolt.DB, volumeName string) error {
	// uniqueness of id is guaranteed by tx.CreateBucket refusing to
	// create the per-volume buckets on collision. this leads to an
	// ugly error, but it's boil-the-oceans rare
	id, err := RandomVolumeID()
	if err != nil {
		return err
	}
	err = db.Update(func(tx *bolt.Tx) error {

		{
			bucket := tx.Bucket(bucketVolName)
			key := []byte(volumeName)
			exists := bucket.Get(key)
			if exists != nil {
				return errors.New("volume name exists already")
			}
			var secret [32]byte
			if _, err := rand.Read(secret[:]); err != nil {
				return err
			}
			volConf := wire.VolumeConfig{
				VolumeID: id.Bytes(),
				Storage: wire.KV{
					Local: &wire.KV_Local{
						Secret: secret[:],
					},
				},
			}
			buf, err := proto.Marshal(&volConf)
			if err != nil {
				return err
			}
			err = bucket.Put(key, buf)
			if err != nil {
				return err
			}
		}

		bucket := tx.Bucket(bucketVolume)
		if bucket, err = bucket.CreateBucket(id.Bytes()); err != nil {
			return err
		}
		if _, err := bucket.CreateBucket(bucketDir); err != nil {
			return err
		}
		if _, err := bucket.CreateBucket(bucketInode); err != nil {
			return err
		}
		if _, err := bucket.CreateBucket(bucketSnap); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (f *Volume) Init(req *fuse.InitRequest, resp *fuse.InitResponse, intr fs.Intr) fuse.Error {
	resp.MaxReadahead = 32 * 1024 * 1024
	resp.Flags |= fuse.InitAsyncRead
	return nil
}

func (v *Volume) Root() (fs.Node, fuse.Error) {
	d := &dir{
		inode:  tokens.InodeRoot,
		parent: nil,
		fs:     v,
		active: make(map[string]node),
	}
	return d, nil
}

func (*Volume) GenerateInode(parent uint64, name string) uint64 {
	return inodes.Dynamic(parent, name)
}

var _ = fs.FSInodeGenerator(&Volume{})

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
