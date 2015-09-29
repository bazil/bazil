package fs

import (
	"os"

	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type dotBazil struct {
	fs     *Volume
	parent *dir
}

var _ = fs.Node(&dotBazil{})

func (d *dotBazil) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0555
	a.Uid = env.MyUID
	a.Uid = env.MyGID
	return nil
}

var _ = fs.NodeStringLookuper(&dotBazil{})

func (d *dotBazil) Lookup(ctx context.Context, name string) (fs.Node, error) {
	switch name {
	case "pending":
		return &pendingList{}, nil
	default:
		return nil, fuse.ENOENT
	}
}
