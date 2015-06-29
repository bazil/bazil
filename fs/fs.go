package fs

import (
	"log"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/inodes"
	wiresnap "bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type Volume struct {
	db         *db.DB
	volID      db.VolumeID
	chunkStore chunks.Store
	root       *dir
}

var _ = fs.FS(&Volume{})
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
	fs.root = newDir(fs, tokens.InodeRoot, nil, "")
	return fs, nil
}

func (v *Volume) Root() (fs.Node, error) {
	return v.root, nil
}

func (*Volume) GenerateInode(parent uint64, name string) uint64 {
	return inodes.Dynamic(parent, name)
}

var _ = fs.FSInodeGenerator(&Volume{})

// Snapshot records a snapshot of the volume. The Snapshot message
// itself has not been persisted yet.
func (v *Volume) Snapshot(ctx context.Context, tx *db.Tx) (*wiresnap.Snapshot, error) {
	snapshot := &wiresnap.Snapshot{}
	sde, err := v.root.snapshot(ctx, tx)
	if err != nil {
		return nil, err
	}
	snapshot.Contents = sde
	return snapshot, nil
}

type node interface {
	fs.Node

	marshal() (*wire.Dirent, error)
	setName(name string)
}
