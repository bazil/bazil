package fs

import (
	"os"

	"bazil.org/bazil/util/env"
	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"golang.org/x/net/context"
)

type pendingList struct {
}

var _ fs.Node = (*pendingList)(nil)

func (l *pendingList) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0500
	a.Uid = env.MyUID
	a.Gid = env.MyGID
	return nil
}
