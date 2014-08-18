package fs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
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
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/boltdb/bolt"
)

type dir struct {
	fs.NodeRef

	inode  uint64
	name   string
	parent *dir
	fs     *Volume

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
var _ = fs.HandleReadDirer(&dir{})

func (d *dir) getName() string {
	return d.name
}

func (d *dir) setName(name string) {
	d.name = name
}

func (d *dir) Attr() fuse.Attr {
	return fuse.Attr{
		Inode: d.inode,
		Mode:  os.ModeDir | 0755,
		Nlink: 1,
		Uid:   env.MyUID,
		Gid:   env.MyGID,
	}
}

func (d *dir) Lookup(name string, intr fs.Intr) (fs.Node, fuse.Error) {
	if d.inode == 1 && name == ".snap" {
		return &listSnaps{
			fs:      d.fs,
			rootDir: d,
		}, nil
	}

	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	if child, ok := d.active[name]; ok {
		return child, nil
	}

	key := pathToKey(d.inode, name)
	var buf []byte
	err := d.fs.db.View(func(tx *bolt.Tx) error {
		bucket := d.fs.bucket(tx).Bucket(bucketDir)
		if bucket == nil {
			return errors.New("dir bucket missing")
		}
		buf = bucket.Get(key)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if buf == nil {
		return nil, fuse.ENOENT
	}
	de, err := d.unmarshalDirent(buf)
	if err != nil {
		return nil, fmt.Errorf("dirent unmarshal problem: %v", err)
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

func (d *dir) ReadDir(intr fs.Intr) ([]fuse.Dirent, fuse.Error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

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
func (d *dir) save(tx *bolt.Tx, name string, n node) error {
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

func (d *dir) Create(req *fuse.CreateRequest, resp *fuse.CreateResponse, intr fs.Intr) (fs.Node, fs.Handle, fuse.Error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

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

			return d.save(tx, req.Name, child)
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
	if have, ok := d.active[name]; ok {
		// have something by that name
		if have == child {
			// has not been overwritten
			delete(d.active, name)
		}
	}
}

func (d *dir) Forget() {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()
	if d.parent == nil {
		// root dir, don't keep track
		return
	}
	d.parent.forgetChild(d.name, d)
}

func (d *dir) Mkdir(req *fuse.MkdirRequest, intr fs.Intr) (fs.Node, fuse.Error) {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

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
		return d.save(tx, req.Name, child)
		// TODO clean up active on error
	})
	if err != nil {
		if err == inodes.OutOfInodes {
			return nil, fuse.Errno(syscall.ENOSPC)
		}
		return nil, err
	}
	return child, nil
}

func (d *dir) Remove(req *fuse.RemoveRequest, intr fs.Intr) fuse.Error {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

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

func (d *dir) Rename(req *fuse.RenameRequest, newDir fs.Node, intr fs.Intr) fuse.Error {
	d.fs.mu.Lock()
	defer d.fs.mu.Unlock()

	// if you ever change this, also guard against renaming into
	// special directories like .snap; check type of newDir is *dir
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
func (d *dir) snapshot(tx *bolt.Tx, out *wiresnap.Dir, intr fs.Intr) error {
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
			err = child.snapshot(tx, sde.Type.Dir, intr)
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
