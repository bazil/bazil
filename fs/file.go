package fs

import (
	"io"
	"log"

	"bazil.org/bazil/cas/blobs"
	wirecas "bazil.org/bazil/cas/wire"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/boltdb/bolt"
)

type file struct {
	inode  uint64
	name   string
	parent *dir
	blob   *blobs.Blob

	// when was this entry last changed
	// TODO: written time.Time
}

var _ = node(&file{})

func (f *file) getName() string {
	return f.name
}

func (f *file) marshal() (*wire.Dirent, error) {
	de := &wire.Dirent{
		Inode: f.inode,
	}
	manifest, err := f.blob.Save()
	if err != nil {
		return nil, err
	}
	de.Type.File = &wire.File{
		Manifest: wirecas.FromBlob(manifest),
	}
	return de, nil
}

func (f *file) Attr() fuse.Attr {
	return fuse.Attr{
		Inode: f.inode,
		Mode:  0644,
		Nlink: 1,
		Uid:   env.MyUID,
		Gid:   env.MyGID,
	}
}

func (f *file) Write(req *fuse.WriteRequest, resp *fuse.WriteResponse, intr fs.Intr) fuse.Error {
	n, err := f.blob.WriteAt(req.Data, req.Offset)
	resp.Size = n
	if err != nil {
		log.Printf("write error: %v", err)
		return fuse.EIO
	}
	return nil
}

func (f *file) Flush(req *fuse.FlushRequest, intr fs.Intr) fuse.Error {
	// TODO only if dirty
	err := f.parent.fs.db.Update(func(tx *bolt.Tx) error {
		return f.parent.save(tx, f)
	})
	return err
}

const maxInt64 = 9223372036854775807

func (f *file) Read(req *fuse.ReadRequest, resp *fuse.ReadResponse, intr fs.Intr) fuse.Error {
	if req.Offset < 0 {
		panic("unreachable")
	}
	if req.Offset > maxInt64 {
		log.Printf("offset is past int64 max: %d", req.Offset)
		return fuse.EIO
	}
	resp.Data = resp.Data[:req.Size]
	n, err := f.blob.ReadAt(resp.Data, int64(req.Offset))
	if err != nil && err != io.EOF {
		log.Printf("read error: %v", err)
		return fuse.EIO
	}
	resp.Data = resp.Data[:n]

	return nil
}
