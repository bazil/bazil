package fs

import (
	"encoding/binary"
	"log"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/inodes"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type Volume struct {
	db         *db.DB
	volID      db.VolumeID
	chunkStore chunks.Store
}

var _ = fs.FS(&Volume{})
var _ = fs.FSIniter(&Volume{})
var _ = fs.FSInodeGenerator(&Volume{})

var bucketVolume = []byte(tokens.BucketVolume)
var bucketVolName = []byte(tokens.BucketVolName)
var bucketDir = []byte("dir")
var bucketInode = []byte("inode")
var bucketSnap = []byte("snap")
var bucketStorage = []byte("storage")

func (v *Volume) bucket(tx *db.Tx) *db.Volume {
	vv, err := tx.Volumes().GetByVolumeID(&v.volID)
	if err != nil {
		log.Printf("volume has disappeared: %v: %v", &v.volID, err)
	}
	return vv
}

// Open returns a FUSE filesystem instance serving content from the
// given volume. The result can be passed to bazil.org/fuse/fs#Serve
// to start serving file access requests from the kernel.
//
// Caller guarantees volume ID exists at least as long as requests are
// served for this file system.
func Open(db *db.DB, chunkStore chunks.Store, volumeID *db.VolumeID) (*Volume, error) {
	fs := &Volume{}
	fs.db = db
	fs.volID = *volumeID
	fs.chunkStore = chunkStore
	return fs, nil
}

func (f *Volume) Init(ctx context.Context, req *fuse.InitRequest, resp *fuse.InitResponse) error {
	resp.MaxReadahead = 32 * 1024 * 1024
	resp.Flags |= fuse.InitAsyncRead
	return nil
}

func (v *Volume) Root() (fs.Node, error) {
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
	setName(name string)
}
