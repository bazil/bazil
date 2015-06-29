package fs

import (
	"errors"
	"fmt"
	"os"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/snap"
	wiresnap "bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
)

type listSnaps struct {
	fs *Volume
}

var _ = fs.Node(&listSnaps{})
var _ = fs.NodeMkdirer(&listSnaps{})
var _ = fs.NodeStringLookuper(&listSnaps{})
var _ = fs.Handle(&listSnaps{})
var _ = fs.HandleReadDirAller(&listSnaps{})

func (d *listSnaps) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Inode = tokens.InodeSnap
	a.Mode = os.ModeDir | 0755
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	return nil
}

var _ = fs.NodeStringLookuper(&listSnaps{})

func (d *listSnaps) Lookup(ctx context.Context, name string) (fs.Node, error) {
	var ref wire.SnapshotRef
	lookup := func(tx *db.Tx) error {
		bucket := d.fs.bucket(tx).SnapBucket()
		if bucket == nil {
			return errors.New("snapshot bucket missing")
		}
		buf := bucket.Get([]byte(name))
		if buf == nil {
			return fuse.ENOENT
		}
		if err := proto.Unmarshal(buf, &ref); err != nil {
			return fmt.Errorf("corrupt snapshot reference: %q: %v", name, err)
		}
		return nil
	}
	if err := d.fs.db.View(lookup); err != nil {
		return nil, err
	}

	var k cas.Key
	if err := k.UnmarshalBinary(ref.Key); err != nil {
		return nil, fmt.Errorf("corrupt snapshot reference: %q: %v", name, err)
	}

	chunk, err := d.fs.chunkStore.Get(k, "snap", 0)
	if err != nil {
		return nil, fmt.Errorf("cannot fetch snapshot: %v", err)
	}

	var snapshot wiresnap.Snapshot
	err = proto.Unmarshal(chunk.Buf, &snapshot)
	if err != nil {
		return nil, fmt.Errorf("corrupt snapshot: %v: %v", ref.Key, err)
	}

	n, err := snap.Open(d.fs.chunkStore, snapshot.Contents)
	if err != nil {
		return nil, fmt.Errorf("cannot serve snapshot: %v", err)
	}
	return n, nil
}

var _ = fs.NodeMkdirer(&listSnaps{})

// Mkdir takes a snapshot of this volume and records it under the
// given name.
func (d *listSnaps) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	var snapshot = wiresnap.Snapshot{
		Name: req.Name,
	}
	record := func(tx *db.Tx) error {
		sde, err := d.fs.root.snapshot(ctx, tx)
		if err != nil {
			return err
		}
		snapshot.Contents = sde
		return nil
	}
	if err := d.fs.db.View(record); err != nil {
		return nil, fmt.Errorf("cannot record snapshot: %v", err)
	}

	var key cas.Key
	{
		buf, err := proto.Marshal(&snapshot)
		if err != nil {
			return nil, fmt.Errorf("cannot marshal snapshot: %v", err)
		}
		if len(buf) == 0 {
			return nil, errors.New("marshaled snapshot become empty; this is a bug!")
		}

		// store the snapshot as a chunk, for disaster recovery
		key, err = d.fs.chunkStore.Add(&chunks.Chunk{
			Type:  "snap",
			Level: 0,
			Buf:   buf,
		})
		if err != nil {
			return nil, fmt.Errorf("cannot store snapshot: %v", err)
		}
	}

	var ref = wire.SnapshotRef{
		Key: key.Bytes(),
	}
	buf, err := proto.Marshal(&ref)
	if err != nil {
		return nil, fmt.Errorf("cannot marshal snapshot pointer: %v", err)
	}

	add := func(tx *db.Tx) error {
		b := d.fs.bucket(tx).SnapBucket()
		if b == nil {
			return errors.New("snapshot bucket missing")
		}
		return b.Put([]byte(req.Name), buf)
	}
	if err := d.fs.db.Update(add); err != nil {
		return nil, fmt.Errorf("cannot save snapshot pointer: %v", err)
	}

	n, err := snap.Open(d.fs.chunkStore, snapshot.Contents)
	if err != nil {
		return nil, fmt.Errorf("cannot serve snapshot: %v", err)
	}
	return n, nil
}

var _ = fs.HandleReadDirAller(&listSnaps{})

func (d *listSnaps) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	// NOT HOLDING LOCKS, accessing database snapshot ONLY

	var entries []fuse.Dirent
	readDirAll := func(tx *db.Tx) error {
		bucket := d.fs.bucket(tx).SnapBucket()
		if bucket == nil {
			return errors.New("snapshot bucket missing")
		}
		c := bucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			fde := fuse.Dirent{
				Name: string(k),
				Type: fuse.DT_Dir,
			}
			entries = append(entries, fde)
		}
		return nil
	}
	if err := d.fs.db.View(readDirAll); err != nil {
		return nil, err
	}
	return entries, nil
}
