package fs

import (
	"fmt"
	"os"
	"syscall"

	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/clock"
	"bazil.org/bazil/fs/readonly"
	wirepeer "bazil.org/bazil/peer/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/tv42/zbase32"
	"golang.org/x/net/context"
)

type pendingList struct {
	dir *dir
}

var _ fs.Node = (*pendingList)(nil)

func (l *pendingList) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0500
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	return nil
}

var _ fs.NodeStringLookuper = (*pendingList)(nil)

func (l *pendingList) Lookup(ctx context.Context, name string) (fs.Node, error) {
	var child *pendingEntry
	lookup := func(tx *db.Tx) error {
		c := l.dir.fs.bucket(tx).Conflicts().ListAll(l.dir.inode)
		item := c.First()
		if item == nil {
			return fuse.ENOENT
		}
		child = &pendingEntry{
			list: l,
			name: item.Name(),
		}
		return nil
	}
	if err := l.dir.fs.db.View(lookup); err != nil {
		return nil, err
	}
	return child, nil
}

var _ fs.HandleReadDirAller = (*pendingList)(nil)

func (l *pendingList) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent
	readDirAll := func(tx *db.Tx) error {
		conflicts := l.dir.fs.bucket(tx).Conflicts()
		c := conflicts.ListAll(l.dir.inode)
		// the iterator sees every name+clock, not just every name;
		// only pass unique names to results
		prev := ""
		for item := c.First(); item != nil; item = c.Next() {
			name := item.Name()
			if name == prev {
				continue
			}
			prev = name
			fde := fuse.Dirent{
				Name: name,
				Type: fuse.DT_Dir,
			}
			entries = append(entries, fde)
		}
		return nil
	}
	if err := l.dir.fs.db.View(readDirAll); err != nil {
		return nil, err
	}
	return entries, nil
}

type pendingEntry struct {
	list *pendingList
	// name of the dentry these are pending clocks for
	name string
}

var _ fs.Node = (*pendingEntry)(nil)

func (e *pendingEntry) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0500
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	return nil
}

var _ fs.NodeStringLookuper = (*pendingEntry)(nil)

func (e *pendingEntry) Lookup(ctx context.Context, name string) (fs.Node, error) {
	clockBuf, err := zbase32.DecodeString(name)
	if err != nil {
		return nil, fuse.ENOENT
	}

	var de wirepeer.Dirent
	lookup := func(tx *db.Tx) error {
		item := e.list.dir.fs.bucket(tx).Conflicts().Get(e.list.dir.inode, e.name, clockBuf)
		if item == nil {
			return fuse.ENOENT
		}
		if err := item.Dirent(&de); err != nil {
			return err
		}
		return nil
	}
	if err := e.list.dir.fs.db.View(lookup); err != nil {
		return nil, err
	}

	var child fs.Node
	switch {
	case de.File != nil:
		manifest, err := de.File.Manifest.ToBlob("file")
		if err != nil {
			return nil, err
		}
		// TODO pass in mode and other metadata
		f, err := readonly.NewFile(e.list.dir.fs.chunkStore, manifest)
		if err != nil {
			return nil, err
		}
		child = f

	default:
		return nil, fmt.Errorf("unsupported pending direntry type: %#v", de)
	}

	return child, nil
}

var _ fs.HandleReadDirAller = (*pendingEntry)(nil)

func (e *pendingEntry) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent
	readDirAll := func(tx *db.Tx) error {
		conflicts := e.list.dir.fs.bucket(tx).Conflicts()
		cursor := conflicts.List(e.list.dir.inode, e.name)
		for item := cursor.First(); item != nil; item = cursor.Next() {
			c, err := item.Clock()
			if err != nil {
				return err
			}
			clockBuf, err := c.MarshalBinary()
			if err != nil {
				return err
			}
			name := zbase32.EncodeToString(clockBuf)
			fde := fuse.Dirent{
				Name: name,
				Type: fuse.DT_Dir,
			}
			entries = append(entries, fde)
		}
		return nil
	}
	if err := e.list.dir.fs.db.View(readDirAll); err != nil {
		return nil, err
	}
	return entries, nil
}

var _ fs.NodeRemover = (*pendingEntry)(nil)

func (e *pendingEntry) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	if req.Dir {
		return fuse.Errno(syscall.ENOTDIR)
	}

	clockBuf, err := zbase32.DecodeString(req.Name)
	if err != nil {
		return fuse.ENOENT
	}
	var loser clock.Clock
	if err := loser.UnmarshalBinary(clockBuf); err != nil {
		return fuse.ENOENT
	}

	// TODO we're assuming the main entry is not deleted (or that it
	// still has a tombstone)?

	update := func(tx *db.Tx) error {
		bucket := e.list.dir.fs.bucket(tx)
		if err := bucket.Conflicts().Delete(e.list.dir.inode, e.name, clockBuf); err != nil {
			return err
		}

		clocks := bucket.Clock()
		c, err := clocks.Get(e.list.dir.inode, e.name)
		if err != nil {
			return err
		}
		c.ResolveOurs(&loser)
		if err := clocks.Put(e.list.dir.inode, e.name, c); err != nil {
			return err
		}
		return nil
	}
	if err := e.list.dir.fs.db.Update(update); err != nil {
		return err
	}

	return nil
}
