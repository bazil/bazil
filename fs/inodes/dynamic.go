package inodes

import (
	"bazil.org/fuse/fs"

	"bazil.org/bazil/tokens"
)

// Dynamic returns a dynamic inode. The result is guaranteed to never
// collide with inodes returned from Allocate.
func Dynamic(parent uint64, name string) uint64 {
	inode := fs.GenerateDynamicInode(parent, name)
	inode &^= tokens.InodeKindMask
	inode |= tokens.InodeKindDynamic
	return inode
}
