package fs

import (
	"io"
	"log"
	"sync"

	"bazil.org/bazil/cas/blobs"
	wirecas "bazil.org/bazil/cas/wire"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type file struct {
	fs.NodeRef

	inode  uint64
	parent *dir

	// mu protects the fields below.
	mu sync.Mutex

	name string
	blob *blobs.Blob

	// when was this entry last changed
	// TODO: written time.Time
}

var _ = node(&file{})
var _ = fs.Node(&file{})
var _ = fs.NodeForgetter(&file{})
var _ = fs.NodeOpener(&file{})
var _ = fs.NodeSetattrer(&file{})
var _ = fs.NodeFsyncer(&file{})
var _ = fs.HandleFlusher(&file{})
var _ = fs.HandleReader(&file{})
var _ = fs.HandleWriter(&file{})

func (f *file) setName(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.name = name
}

func (f *file) marshal() (*wire.Dirent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	de := &wire.Dirent{
		Inode: f.inode,
	}
	manifest, err := f.blob.Save()
	if err != nil {
		return nil, err
	}
	de.File = &wire.File{
		Manifest: wirecas.FromBlob(manifest),
	}
	return de, nil
}

func (f *file) Attr(a *fuse.Attr) {
	f.mu.Lock()
	defer f.mu.Unlock()

	a.Inode = f.inode
	a.Mode = 0644
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	a.Size = f.blob.Size()
}

func (f *file) Forget() {
	f.parent.forgetChild(f.name, f)
}

func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// allow kernel to use buffer cache
	resp.Flags &^= fuse.OpenDirectIO
	return f, nil
}

func (f *file) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	n, err := f.blob.WriteAt(req.Data, req.Offset)
	resp.Size = n
	if err != nil {
		log.Printf("write error: %v", err)
		return fuse.EIO
	}
	return nil
}

func (f *file) flush(ctx context.Context) error {
	// TODO only if dirty
	err := f.parent.fs.db.Update(func(tx *db.Tx) error {
		return f.parent.save(tx, f.name, f)
	})
	return err
}

func (f *file) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	return f.flush(ctx)
}

const maxInt64 = 9223372036854775807

func (f *file) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

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

func (f *file) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	valid := req.Valid
	if valid.Size() {
		err := f.blob.Truncate(req.Size)
		if err != nil {
			return err
		}
		valid &^= fuse.SetattrSize
	}

	// things we don't need to explicitly handle
	valid &^= fuse.SetattrLockOwner | fuse.SetattrHandle

	if valid != 0 {
		// don't let an unhandled operation slip by without error
		log.Printf("Setattr did not handle %v", valid)
		return fuse.ENOSYS
	}
	return nil
}

func (f *file) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
	// flush forces writes to backing stores; we don't current
	// differentiate between the backing stores writing vs syncing.
	return f.flush(ctx)
}
