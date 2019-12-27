package snap

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/fs/readonly"
	"bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// Serve this snapshot with FUSE, with this object store.
func Open(chunkStore chunks.Store, de *wire.Dirent) (fusefs.Node, error) {
	switch de := de.Type.(type) {
	case *wire.Dirent_File:
		manifest, err := de.File.Manifest.ToBlob("file")
		if err != nil {
			return nil, err
		}
		child, err := readonly.NewFile(chunkStore, manifest)
		if err != nil {
			return nil, fmt.Errorf("snap file open error: %v", err)
		}
		return child, nil

	case *wire.Dirent_Dir:
		manifest, err := de.Dir.Manifest.ToBlob("dir")
		if err != nil {
			return nil, err
		}
		blob, err := blobs.Open(chunkStore, manifest)
		if err != nil {
			return nil, err
		}
		child := fuseDir{
			chunkStore: chunkStore,
			blob:       blob,
			align:      de.Dir.Align,
		}
		return child, nil

	default:
		return nil, fmt.Errorf("unknown entry in tree, %v", de)
	}
}

type fuseDir struct {
	chunkStore chunks.Store
	blob       *blobs.Blob
	align      uint32
}

var _ fusefs.Node = fuseDir{}
var _ fusefs.NodeStringLookuper = fuseDir{}
var _ fusefs.NodeCreater = fuseDir{}
var _ fusefs.Handle = fuseDir{}
var _ fusefs.HandleReadDirAller = fuseDir{}

func (d fuseDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0555
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	return nil
}

func (d fuseDir) Lookup(ctx context.Context, name string) (fusefs.Node, error) {
	r, err := NewReader(d.blob.IO(ctx), d.align)
	if err != nil {
		return nil, err
	}
	de, err := r.Lookup(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fuse.ENOENT
		}
		return nil, fmt.Errorf("snap lookup error: %v", err)
	}
	return Open(d.chunkStore, de)
}

func (d fuseDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	r, err := NewReader(d.blob.IO(ctx), d.align)
	if err != nil {
		return nil, err
	}
	var list []fuse.Dirent
	it := r.Iter()
	var de *wire.Dirent
	for {
		de, err = it.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return list, fmt.Errorf("snap readdir error: %v", err)
		}
		fde := fuse.Dirent{
			Name: de.Name,
		}
		switch de.Type.(type) {
		case *wire.Dirent_File:
			fde.Type = fuse.DT_File
		case *wire.Dirent_Dir:
			fde.Type = fuse.DT_Dir
		}
		list = append(list, fde)
	}
	return list, nil
}

func (d fuseDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fusefs.Node, fusefs.Handle, error) {
	return nil, nil, fuse.Errno(syscall.EROFS)
}
