package snap

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

// Serve this snapshot with FUSE, with this object store.
func Open(chunkStore chunks.Store, dir *wire.Dir) (fusefs.Node, error) {
	manifest, err := dir.Manifest.ToBlob("dir")
	if err != nil {
		return nil, err
	}
	blob, err := blobs.Open(chunkStore, manifest)
	if err != nil {
		return nil, err
	}
	r, err := NewReader(blob, dir.Align)
	if err != nil {
		return nil, err
	}
	node := fuseDir{
		chunkStore: chunkStore,
		reader:     r,
	}
	return node, nil
}

type fuseDir struct {
	chunkStore chunks.Store
	reader     *Reader
}

var _ = fusefs.Node(fuseDir{})
var _ = fusefs.NodeStringLookuper(fuseDir{})
var _ = fusefs.NodeCreater(fuseDir{})
var _ = fusefs.Handle(fuseDir{})
var _ = fusefs.HandleReadDirAller(fuseDir{})

func (d fuseDir) Attr(a *fuse.Attr) {
	a.Mode = os.ModeDir | 0555
	a.Uid = env.MyUID
	a.Gid = env.MyGID
}

const _MAX_INT64 = 9223372036854775807

func (d fuseDir) Lookup(ctx context.Context, name string) (fusefs.Node, error) {
	de, err := d.reader.Lookup(name)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fuse.ENOENT
		}
		return nil, fmt.Errorf("snap lookup error: %v", err)
	}

	switch {
	case de.Type.File != nil:
		manifest, err := de.Type.File.Manifest.ToBlob("file")
		if err != nil {
			return nil, err
		}
		blob, err := blobs.Open(d.chunkStore, manifest)
		if err != nil {
			return nil, fmt.Errorf("snap file blob open error: %v", err)
		}
		child := fuseFile{
			rat: blob,
			de:  de,
		}
		return child, nil

	case de.Type.Dir != nil:
		child, err := Open(d.chunkStore, de.Type.Dir)
		if err != nil {
			return nil, fmt.Errorf("snap dir FUSE serving error: %v", err)
		}
		return child, nil

	default:
		return nil, fmt.Errorf("unknown entry in tree, %v", de.Type.GetValue())
	}
}

func (d fuseDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var list []fuse.Dirent
	it := d.reader.Iter()
	var de *wire.Dirent
	var err error
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
		if de.Type.File != nil {
			fde.Type = fuse.DT_File
		} else if de.Type.Dir != nil {
			fde.Type = fuse.DT_Dir
		}
		list = append(list, fde)
	}
	return list, nil
}

func (d fuseDir) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fusefs.Node, fusefs.Handle, error) {
	return nil, nil, fuse.Errno(syscall.EROFS)
}

func stat_blocks(size uint64) uint64 {
	r := size / 512
	if size%512 > 0 {
		r++
	}
	return r
}
