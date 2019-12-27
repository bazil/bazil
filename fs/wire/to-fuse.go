package wire

import (
	"bazil.org/fuse"
)

// GetFUSEDirent returns a populated fuse.Dirent
func (d *Dirent) GetFUSEDirent(name string) fuse.Dirent {
	fde := fuse.Dirent{
		Inode: d.Inode,
		Name:  name,
	}
	switch d.Type.(type) {
	case *Dirent_File:
		fde.Type = fuse.DT_File

	case *Dirent_Dir:
		fde.Type = fuse.DT_Dir
	}
	return fde
}
