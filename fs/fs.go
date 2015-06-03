package fs

import (
	"log"
	"sync"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/clock"
	"bazil.org/bazil/fs/inodes"
	wiresnap "bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type Volume struct {
	db         *db.DB
	volID      db.VolumeID
	pubKey     peer.PublicKey
	chunkStore chunks.Store
	root       *dir

	epoch struct {
		mu sync.Mutex
		// Epoch is a logical clock keeping track of file mutations. It
		// increments for every outgoing (dirty) sync of this volume.
		ticks clock.Epoch
		// Have changes been made since epoch ticked?
		dirty bool
	}
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
func Open(db *db.DB, chunkStore chunks.Store, volumeID *db.VolumeID, pubKey *peer.PublicKey) (*Volume, error) {
	fs := &Volume{}
	fs.db = db
	fs.volID = *volumeID
	fs.pubKey = *pubKey
	fs.chunkStore = chunkStore
	fs.root = newDir(fs, tokens.InodeRoot, nil, "")
	// assume we crashed, to be safe
	fs.epoch.dirty = true
	if err := fs.db.View(fs.initFromDB); err != nil {
		return nil, err
	}
	return fs, nil
}

func (v *Volume) initFromDB(tx *db.Tx) error {
	epoch, err := v.bucket(tx).Epoch()
	if err != nil {
		return err
	}
	v.epoch.ticks = epoch
	return nil
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

// caller is responsible for locking
//
// TODO nextEpoch only needs to tick if the volume is seeing mutation;
// unmounted is safe?
func (v *Volume) nextEpoch(vb *db.Volume) error {
	if !v.epoch.dirty {
		return nil
	}
	n, err := vb.NextEpoch()
	if err != nil {
		return err
	}
	v.epoch.ticks = n
	v.epoch.dirty = false
	return nil
}

func (v *Volume) dirtyEpoch() clock.Epoch {
	v.epoch.mu.Lock()
	defer v.epoch.mu.Unlock()
	v.epoch.dirty = true
	return v.epoch.ticks
}

func (v *Volume) cleanEpoch() (clock.Epoch, error) {
	v.epoch.mu.Lock()
	defer v.epoch.mu.Unlock()
	if !v.epoch.dirty {
		return v.epoch.ticks, nil
	}
	inc := func(tx *db.Tx) error {
		vb := v.bucket(tx)
		if err := v.nextEpoch(vb); err != nil {
			return err
		}
		return nil
	}
	if err := v.db.Update(inc); err != nil {
		return 0, err
	}
	return v.epoch.ticks, nil
}

type node interface {
	fs.Node

	marshal() (*wire.Dirent, error)
	setName(name string)
}
