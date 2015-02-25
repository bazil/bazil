package fs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"

	"bazil.org/bazil/cas/blobs"
	wirecas "bazil.org/bazil/cas/wire"
	"bazil.org/bazil/fs/inodes"
	"bazil.org/bazil/fs/snap"
	wiresnap "bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/boltdb/bolt"
	"github.com/gogo/protobuf/proto"
	"golang.org/x/net/context"
)

type dir struct {
	fs.NodeRef

	inode  uint64
	parent *dir
	fs     *Volume

	// mu protects the fields below.
	mu sync.Mutex

	name string

	// each in-memory child, so we can return the same node on
	// multiple Lookups and know what to do on .save()
	//
	// each child also stores its own name; if the value in the child,
	// looked up in this map, does not equal the child, that means the
	// child has been unlinked
	active map[string]node
}

var _ = node(&dir{})
var _ = fs.Node(&dir{})
var _ = fs.NodeCreater(&dir{})
var _ = fs.NodeForgetter(&dir{})
var _ = fs.NodeMkdirer(&dir{})
var _ = fs.NodeRemover(&dir{})
var _ = fs.NodeRenamer(&dir{})
var _ = fs.NodeStringLookuper(&dir{})
var _ = fs.HandleReadDirAller(&dir{})

func (d *dir) setName(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.name = name
}

func (d *dir) Attr(a *fuse.Attr) {
	a.Inode = d.inode
	a.Mode = os.ModeDir | 0755
	a.Uid = env.MyUID
	a.Gid = env.MyGID
}

func (d *dir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	if d.inode == 1 && name == ".snap" {
		return &listSnaps{
			fs:      d.fs,
			rootDir: d,
		}, nil
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if child, ok := d.active[name]; ok {
		return child, nil
	}

	var de *wire.Dirent
	key := pathToKey(d.inode, name)
	err := d.fs.db.View(func(tx *bolt.Tx) error {
		bucket := d.fs.bucket(tx).Bucket(bucketDir)
		if bucket == nil {
			return errors.New("dir bucket missing")
		}
		buf := bucket.Get(key)
		if buf == nil {
			return fuse.ENOENT
		}
		var err error
		de, err = d.unmarshalDirent(buf)
		if err != nil {
			return fmt.Errorf("dirent unmarshal problem: %v", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	child, err := d.reviveNode(de, name)
	if err != nil {
		return nil, fmt.Errorf("dirent node unmarshal problem: %v", err)
	}
	d.active[name] = child
	return child, nil
}

func (d *dir) unmarshalDirent(buf []byte) (*wire.Dirent, error) {
	var de wire.Dirent
	err := proto.Unmarshal(buf, &de)
	if err != nil {
		return nil, err
	}
	return &de, nil
}

func (d *dir) reviveDir(de *wire.Dirent, name string) (*dir, error) {
	if de.Type.Dir == nil {
		return nil, fmt.Errorf("tried to revive non-directory as directory: %v", de.GetValue())
	}
	child := &dir{
		inode:  de.Inode,
		name:   name,
		parent: d,
		fs:     d.fs,
		active: make(map[string]node),
	}
	return child, nil
}

func (d *dir) reviveNode(de *wire.Dirent, name string) (node, error) {
	switch {
	case de.Type.Dir != nil:
		return d.reviveDir(de, name)

	case de.Type.File != nil:
		manifest := de.Type.File.Manifest.ToBlob("file")
		blob, err := blobs.Open(d.fs.chunkStore, manifest)
		if err != nil {
			return nil, err
		}
		child := &file{
			inode:  de.Inode,
			name:   name,
			parent: d,
			blob:   blob,
		}
		return child, nil
	}

	return nil, fmt.Errorf("dirent unknown type: %v", de.GetValue())
}

func (d *dir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var entries []fuse.Dirent
	err := d.fs.db.View(func(tx *bolt.Tx) error {
		bucket := d.fs.bucket(tx).Bucket(bucketDir)
		if bucket == nil {
			return errors.New("dir bucket missing")
		}
		c := bucket.Cursor()
		prefix := pathToKey(d.inode, "")
		for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
			if !bytes.HasPrefix(k, prefix) {
				// past the end of the directory
				break
			}

			name := string(k[len(prefix):])
			de, err := d.unmarshalDirent(v)
			if err != nil {
				return fmt.Errorf("readdir error: %v", err)
			}
			fde := de.GetFUSEDirent(name)
			entries = append(entries, fde)
		}
		return nil
	})
	return entries, err
}

// caller does locking
func (d *dir) saveInternal(tx *bolt.Tx, name string, n node) error {
	if have, ok := d.active[name]; !ok || have != n {
		// unlinked
		return nil
	}

	de, err := n.marshal()
	if err != nil {
		return fmt.Errorf("node save error: %v", err)
	}

	buf, err := proto.Marshal(de)
	if err != nil {
		return fmt.Errorf("Dirent marshal error: %v", err)
	}

	key := pathToKey(d.inode, name)
	bucket := d.fs.bucket(tx).Bucket(bucketDir)
	if bucket == nil {
		return errors.New("dir bucket missing")
	}
	err = bucket.Put(key, buf)
	if err != nil {
		return fmt.Errorf("db write error: %v", err)
	}
	return nil
}

func (d *dir) marshal() (*wire.Dirent, error) {
	de := &wire.Dirent{
		Inode: d.inode,
	}
	de.Type.Dir = &wire.Dir{}
	return de, nil
}

func (d *dir) save(tx *bolt.Tx, name string, n node) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.saveInternal(tx, name, n)
}

func (d *dir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// TODO check for duplicate name

	switch req.Mode & os.ModeType {
	case 0:
		var child node
		err := d.fs.db.Update(func(tx *bolt.Tx) error {
			bucket := d.fs.bucket(tx).Bucket(bucketInode)
			if bucket == nil {
				return errors.New("inode bucket is missing")
			}
			inode, err := inodes.Allocate(bucket)
			if err != nil {
				return err
			}

			manifest := blobs.EmptyManifest("file")
			blob, err := blobs.Open(d.fs.chunkStore, manifest)
			if err != nil {
				return fmt.Errorf("blob open problem: %v", err)
			}
			child = &file{
				inode:  inode,
				name:   req.Name,
				parent: d,
				blob:   blob,
			}
			d.active[req.Name] = child

			return d.saveInternal(tx, req.Name, child)
			// TODO clean up active on error
		})
		if err != nil {
			return nil, nil, err
		}
		return child, child, nil
	default:
		return nil, nil, fuse.EPERM
	}
}

// caller does locking
func (d *dir) forgetChild(name string, child node) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if have, ok := d.active[name]; ok {
		// have something by that name
		if have == child {
			// has not been overwritten
			delete(d.active, name)
		}
	}
}

func (d *dir) Forget() {
	if d.parent == nil {
		// root dir, don't keep track
		return
	}
	d.parent.forgetChild(d.name, d)
}

func (d *dir) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// TODO handle req.Mode

	var child node
	err := d.fs.db.Update(func(tx *bolt.Tx) error {
		bucket := d.fs.bucket(tx).Bucket(bucketInode)
		if bucket == nil {
			return errors.New("inode bucket is missing")
		}
		inode, err := inodes.Allocate(bucket)
		if err != nil {
			return err
		}
		child = &dir{
			inode:  inode,
			name:   req.Name,
			parent: d,
			fs:     d.fs,
			active: make(map[string]node),
		}
		d.active[req.Name] = child
		return d.saveInternal(tx, req.Name, child)
		// TODO clean up active on error
	})
	if err != nil {
		if err == inodes.ErrOutOfInodes {
			return nil, fuse.Errno(syscall.ENOSPC)
		}
		return nil, err
	}
	return child, nil
}

func (d *dir) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	key := pathToKey(d.inode, req.Name)
	err := d.fs.db.Update(func(tx *bolt.Tx) error {
		bucket := d.fs.bucket(tx).Bucket(bucketDir)
		if bucket == nil {
			return errors.New("dir bucket missing")
		}

		// does it exist? can short-circuit existence check if active
		if _, ok := d.active[req.Name]; !ok {
			if bucket.Get(key) == nil {
				return fuse.ENOENT
			}
		}

		err := bucket.Delete(key)
		if err != nil {
			return err
		}
		delete(d.active, req.Name)

		// TODO free inode
		return nil
	})
	return err
}

func (d *dir) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// if you ever change this, also guard against renaming into
	// special directories like .snap; check type of newDir is *dir
	//
	// also worry about deadlocks
	if newDir != d {
		return fuse.Errno(syscall.EXDEV)
	}

	kOld := pathToKey(d.inode, req.OldName)
	kNew := pathToKey(d.inode, req.NewName)

	// the file getting overwritten
	var loserInode uint64

	err := d.fs.db.Update(func(tx *bolt.Tx) error {
		bucket := d.fs.bucket(tx).Bucket(bucketDir)
		if bucket == nil {
			return errors.New("dir bucket missing")
		}

		// TODO don't need to load from db if req.OldName is in active.
		// instead, save active state if we have it; call .save() not this
		// kludge
		bufOld := bucket.Get(kOld)
		if bufOld == nil {
			return fuse.ENOENT
		}

		{
			// TODO don't need to load from db if req.NewName is in active
			bufLoser := bucket.Get(kNew)
			if bufLoser != nil {
				// overwriting
				deLoser, err := d.unmarshalDirent(bufLoser)
				if err != nil {
					return fmt.Errorf("dirent unmarshal problem: %v", err)
				}
				loserInode = deLoser.Inode
			}
		}

		if err := bucket.Put(kNew, bufOld); err != nil {
			return err
		}
		if err := bucket.Delete(kOld); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// if the source inode is active, record its new name
	if nodeOld, ok := d.active[req.OldName]; ok {
		nodeOld.setName(req.NewName)
		delete(d.active, req.OldName)
		d.active[req.NewName] = nodeOld
	}

	if loserInode > 0 {
		// TODO free loser inode
	}

	return nil
}

// snapshot records a snapshot of the directory and stores it in wde
func (d *dir) snapshot(ctx context.Context, tx *bolt.Tx, out *wiresnap.Dir) error {
	// NOT HOLDING THE LOCK, accessing database snapshot ONLY

	// TODO move bucket lookup to caller?
	bucket := d.fs.bucket(tx).Bucket(bucketDir)
	if bucket == nil {
		return errors.New("dir bucket missing")
	}

	manifest := blobs.EmptyManifest("dir")
	blob, err := blobs.Open(d.fs.chunkStore, manifest)
	if err != nil {
		return err
	}
	w := snap.NewWriter(blob)

	c := bucket.Cursor()
	prefix := pathToKey(d.inode, "")
	for k, v := c.Seek(prefix); k != nil; k, v = c.Next() {
		if !bytes.HasPrefix(k, prefix) {
			// past the end of the directory
			break
		}

		name := string(k[len(prefix):])
		de, err := d.unmarshalDirent(v)
		if err != nil {
			return err
		}
		sde := wiresnap.Dirent{
			Name: name,
		}
		switch {
		case de.Type.File != nil:
			// TODO d.reviveNode would do blobs.Open and that's a bit
			// too much work; rework the apis
			sde.Type.File = &wiresnap.File{
				Manifest: de.Type.File.Manifest,
			}
		case de.Type.Dir != nil:
			child, err := d.reviveDir(de, name)
			if err != nil {
				return err
			}
			sde.Type.Dir = &wiresnap.Dir{}
			err = child.snapshot(ctx, tx, sde.Type.Dir)
			if err != nil {
				return err
			}
		default:
			return errors.New("TODO")
		}
		err = w.Add(&sde)
		if err != nil {
			return err
		}
	}

	manifest, err = blob.Save()
	if err != nil {
		return err
	}
	out.Manifest = wirecas.FromBlob(manifest)
	out.Align = w.Align()
	return nil
}
