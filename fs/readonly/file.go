package readonly

import (
	"fmt"
	"io"
	"syscall"

	"golang.org/x/net/context"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
)

// NewFile opens a file for read-only access.
//
// TODO pass in metadata.
func NewFile(chunkStore chunks.Store, manifest *blobs.Manifest) (fusefs.Node, error) {
	blob, err := blobs.Open(chunkStore, manifest)
	if err != nil {
		return nil, fmt.Errorf("blob open error: %v", err)
	}
	child := &roFile{
		blob: blob,
	}
	return child, nil
}

type roFile struct {
	blob *blobs.Blob
}

var _ fusefs.Node = (*roFile)(nil)

func statBlocks(size uint64) uint64 {
	r := size / 512
	if size%512 > 0 {
		r++
	}
	return r
}

func (f *roFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0444
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	a.Size = f.blob.Size()
	// a.Mtime = e.Meta.Written.UTC()
	// a.Ctime = e.Meta.Written.UTC()
	// a.Crtime = e.Meta.Written.UTC()
	a.Blocks = statBlocks(a.Size) // TODO .Space?
	return nil
}

var _ fusefs.NodeOpener = (*roFile)(nil)

func (f *roFile) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fusefs.Handle, error) {
	if !req.Flags.IsReadOnly() {
		return nil, fuse.Errno(syscall.EACCES)
	}

	return f, nil
}

var _ fusefs.Handle = (*roFile)(nil)
var _ fusefs.HandleReader = (*roFile)(nil)

func (f *roFile) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	// TODO ReadAt is more strict about not giving partial reads
	// than we care about, but i like the lack of cursor
	resp.Data = resp.Data[0:cap(resp.Data)]
	n, err := f.blob.ReadAt(resp.Data, req.Offset)
	resp.Data = resp.Data[:n]
	if err != nil && err != io.EOF {
		return err
	}

	return nil
}
