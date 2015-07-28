package fs

import (
	"io"
	"log"
	"sync"
	"syscall"

	"bazil.org/bazil/cas/blobs"
	wirecas "bazil.org/bazil/cas/wire"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type dirtiness int

const (
	clean dirtiness = iota
	dirty
	writing
)

type file struct {
	inode  uint64
	parent *dir

	// mu protects the fields below.
	mu sync.Mutex

	name    string
	blob    *blobs.Blob
	dirty   dirtiness
	handles uint32

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
var _ = fs.HandleReleaser(&file{})

func (f *file) setName(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.name = name
}

func (f *file) marshalInternal() (*wire.Dirent, error) {
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

func (f *file) marshal() (*wire.Dirent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.marshalInternal()
}

func (f *file) Attr(ctx context.Context, a *fuse.Attr) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	a.Inode = f.inode
	a.Mode = 0644
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	a.Size = f.blob.Size()
	return nil
}

func (f *file) Forget() {
	f.mu.Lock()
	name := f.name
	f.mu.Unlock()

	f.parent.forgetChild(name, f)
}

func (f *file) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// allow kernel to use buffer cache
	resp.Flags &^= fuse.OpenDirectIO
	f.mu.Lock()
	defer f.mu.Unlock()
	tmp := f.handles + 1
	if tmp == 0 {
		return nil, fuse.Errno(syscall.ENFILE)
	}
	f.handles = tmp
	return f, nil
}

func (f *file) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.dirty = dirty

	n, err := f.blob.WriteAt(req.Data, req.Offset)
	resp.Size = n
	if err != nil {
		log.Printf("write error: %v", err)
		return fuse.EIO
	}
	return nil
}

func (f *file) flush(ctx context.Context) error {
	f.mu.Lock()
	locked := true
	defer func() {
		if locked {
			f.mu.Unlock()
		}
	}()

	// only if dirty
	if f.dirty == clean {
		return nil
	}
	f.dirty = writing

	de, err := f.marshalInternal()
	if err != nil {
		return err
	}

	f.mu.Unlock()
	locked = false

	save := func(tx *db.Tx) error {
		return f.parent.save(tx, f.name, de)
	}
	if err := f.parent.fs.db.Update(save); err != nil {
		return err
	}

	f.mu.Lock()
	if f.dirty == writing {
		// was not dirtied in the meanwhile
		f.dirty = clean
	}
	f.mu.Unlock()
	return nil
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

	f.dirty = dirty

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

func (f *file) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	// name will be set to filename if this was the last open handle;
	// this also neatly ignores deleted files
	name := ""
	f.mu.Lock()
	f.handles--
	if f.handles == 0 {
		name = f.name
	}
	f.mu.Unlock()
	if name != "" {
		f.parent.tryResolveConflicts(name)
	}
	return nil
}
