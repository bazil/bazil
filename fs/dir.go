package fs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"syscall"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/fs/inodes"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/boltdb/bolt"
)

type dir struct {
	inode uint64
	name  string
	fs    *Volume
}

var _ = node(&dir{})

func (d *dir) getName() string {
	return d.name
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

func (d *dir) reviveNode(de *wire.Dirent, name string) (node, error) {
	switch {
	case de.Type.Dir != nil:
		child := &dir{
			inode: de.Inode,
			name:  name,
			fs:    d.fs,
		}
		return child, nil

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

func (d *dir) save(tx *bolt.Tx, n node) error {
	de, err := n.marshal()
	if err != nil {
		return fmt.Errorf("node save error: %v", err)
	}

	buf, err := proto.Marshal(de)
	if err != nil {
		return fmt.Errorf("Dirent marshal error: %v", err)
	}

	key := pathToKey(d.inode, n.getName())
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

			return d.save(tx, child)
		})
		if err != nil {
			return nil, nil, err
		}
		return child, child, nil
	default:
		return nil, nil, fuse.EPERM
	}
}

func (d *dir) Mkdir(req *fuse.MkdirRequest, intr fs.Intr) (fs.Node, fuse.Error) {
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
			inode: inode,
			name:  req.Name,
			fs:    d.fs,
		}
		return d.save(tx, child)
	})
	if err != nil {
		if err == inodes.OutOfInodes {
			return nil, fuse.Errno(syscall.ENOSPC)
		}
		return nil, err
	}
	return child, nil
}
