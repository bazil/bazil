package snap

import (
	"io"
	"syscall"

	"bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type fuseFile struct {
	de  *wire.Dirent
	rat io.ReaderAt
}

var _ = fusefs.Node(fuseFile{})
var _ = fusefs.NodeOpener(fuseFile{})
var _ = fusefs.Handle(fuseFile{})
var _ = fusefs.HandleReader(fuseFile{})

func (e fuseFile) Attr(a *fuse.Attr) {
	a.Mode = 0444
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	a.Size = e.de.File.Manifest.Size
	// a.Mtime = e.Meta.Written.UTC()
	// a.Ctime = e.Meta.Written.UTC()
	// a.Crtime = e.Meta.Written.UTC()
	a.Blocks = stat_blocks(e.de.File.Manifest.Size) // TODO .Space?
}

func (e fuseFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fusefs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}

	return e, nil
}

func (e fuseFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// TODO ReadAt is more strict about not giving partial reads
	// than we care about, but i like the lack of cursor
	resp.Data = resp.Data[0:cap(resp.Data)]
	n, err := e.rat.ReadAt(resp.Data, req.Offset)
	resp.Data = resp.Data[:n]
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
