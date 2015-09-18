package snap_test

import (
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/mock"
	wirecas "bazil.org/bazil/cas/wire"
	"bazil.org/bazil/fs/snap"
	"bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/util/tempdir"
	"bazil.org/fuse/fs"
	"bazil.org/fuse/fs/fstestutil"
)

type FS struct {
	root fs.Node
}

var _ fs.FS = (*FS)(nil)

func (f *FS) Root() (fs.Node, error) {
	return f.root, nil
}

func newFS(chunkStore chunks.Store, de *wire.Dirent) (*FS, error) {
	root, err := snap.Open(chunkStore, de)
	if err != nil {
		return nil, err
	}
	return &FS{root: root}, nil
}

var TIME_1 = time.Unix(1361927841, 123456789)

const GREETING = "hello, world\n"

func setup_greeting(t testing.TB, chunkStore chunks.Store) *blobs.Manifest {
	blob, err := blobs.Open(
		chunkStore,
		blobs.EmptyManifest("file"),
	)
	if err != nil {
		t.Fatalf("unexpected blob open error: %v", err)
	}
	_, err = blob.WriteAt([]byte(GREETING), 0)
	if err != nil {
		t.Fatalf("unexpected write error: %v", err)
	}
	manifest, err := blob.Save()
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	return manifest
}

func setup_dir(t testing.TB, chunkStore chunks.Store, dirents []*wire.Dirent) *wire.Dirent {
	blob, err := blobs.Open(
		chunkStore,
		blobs.EmptyManifest("dir"),
	)
	if err != nil {
		t.Fatalf("unexpected blob open error: %v", err)
	}
	w := snap.NewWriter(blob)
	for _, de := range dirents {
		err := w.Add(de)
		if err != nil {
			t.Fatalf("unexpected add error: %v", err)
		}
	}
	manifest, err := blob.Save()
	if err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	var de wire.Dirent
	de.Dir = &wire.Dir{
		Manifest: wirecas.FromBlob(manifest),
	}
	return &de
}

func setup_fs(t *testing.T) fs.FS {
	chunkStore := &mock.InMemory{}

	greeting := setup_greeting(t, chunkStore)
	dir := setup_dir(t, chunkStore, []*wire.Dirent{
		&wire.Dirent{
			Name: "hello",
			File: &wire.File{
				Manifest: wirecas.FromBlob(greeting),
			},
			// Space:   uint64(len(GREETING)),
			// Written: TIME_1,
		},
	})

	filesys, err := newFS(chunkStore, dir)
	if err != nil {
		t.Fatalf("cannot serve snapshot as FUSE: %v", err)
	}
	return filesys
}

func TestHello(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()

	filesys := setup_fs(t)

	mnt, err := fstestutil.MountedT(t, filesys, nil)
	if err != nil {
		t.Fatalf("Mount fail: %v\n", err)
	}
	defer mnt.Close()

	fi, err := os.Stat(mnt.Dir)
	if err != nil {
		t.Fatalf("root getattr failed with %v", err)
	}
	mode := fi.Mode()
	if (mode & os.ModeType) != os.ModeDir {
		t.Errorf("root is not a directory: %#v", fi)
	}
	if mode.Perm() != 0555 {
		t.Errorf("root has weird access mode: %v", mode.Perm())
	}
	switch stat := fi.Sys().(type) {
	case *syscall.Stat_t:
		if stat.Nlink != 1 {
			t.Errorf("root has wrong link count: %v", stat.Nlink)
		}
		if stat.Uid != uint32(syscall.Getuid()) {
			t.Errorf("root has wrong uid: %d", stat.Uid)
		}
		if stat.Gid != uint32(syscall.Getgid()) {
			t.Errorf("root has wrong gid: %d", stat.Gid)
		}
		if stat.Gid != uint32(syscall.Getgid()) {
			t.Errorf("root has wrong gid: %d", stat.Gid)
		}
	}

	f, err := os.Create(path.Join(mnt.Dir, "does-not-exist-yet"))
	if err == nil {
		f.Close()
		t.Errorf("root must be read-only")
	} else if err.(*os.PathError).Err != syscall.EROFS {
		t.Errorf("file create gave bad error: %v", err)
	}

	hello_path := path.Join(mnt.Dir, "hello")
	fi, err = os.Stat(hello_path)
	if err != nil {
		t.Fatalf("hello getattr failed with %v", err)
	}
	if fi.Name() != "hello" {
		t.Errorf("hello has weird name: %q", fi.Name())
	}
	if fi.Size() != 13 {
		t.Errorf("hello has weird size: %v", fi.Size())
	}
	// if fi.ModTime() != TIME_1 {
	// 	t.Errorf("hello has weird time: %v != %v", fi.ModTime(), TIME_1)
	// }
	mode = fi.Mode()
	if (mode & os.ModeType) != 0 {
		t.Errorf("hello is not a file: %#v", fi)
	}
	if mode.Perm() != 0444 {
		t.Errorf("hello has weird access mode: %v", mode.Perm())
	}
	switch stat := fi.Sys().(type) {
	case *syscall.Stat_t:
		if stat.Nlink != 1 {
			t.Errorf("hello has wrong link count: %v", stat.Nlink)
		}
		if stat.Uid != uint32(syscall.Getuid()) {
			t.Errorf("hello has wrong uid: %d", stat.Uid)
		}
		if stat.Gid != uint32(syscall.Getgid()) {
			t.Errorf("hello has wrong gid: %d", stat.Gid)
		}
		if stat.Gid != uint32(syscall.Getgid()) {
			t.Errorf("hello has wrong gid: %d", stat.Gid)
		}
		if stat.Blocks != 1 {
			t.Errorf("hello has weird blockcount: %v", stat.Blocks)
		}
	}

	f, err = os.Create(hello_path)
	if err == nil {
		f.Close()
		t.Errorf("hello must be read-only")
	} else if err.(*os.PathError).Err != syscall.EACCES {
		t.Errorf("hello open for write gave bad error: %v", err)
	}

	f, err = os.Open(hello_path)
	if err != nil {
		t.Fatalf("hello open failed with %v", err)
	}
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("hello read failed with %v", err)
	}
	if string(buf) != GREETING {
		t.Errorf("hello read wrong content: %q", string(buf))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("hello close failed with %v", err)
	}
}

func TestTwoLevels(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()

	setup_fs := func() fs.FS {
		chunkStore := &mock.InMemory{}
		greeting := setup_greeting(t, chunkStore)
		dir1 := setup_dir(t, chunkStore, []*wire.Dirent{
			&wire.Dirent{
				Name: "hello",
				File: &wire.File{
					Manifest: wirecas.FromBlob(greeting),
				},
				// Space:   uint64(len(GREETING)),
				// Written: TIME_1,
			},
		})
		dir1.Name = "second"
		dir2 := setup_dir(t, chunkStore, []*wire.Dirent{dir1})

		filesys, err := newFS(chunkStore, dir2)
		if err != nil {
			t.Fatalf("cannot serve snapshot as FUSE: %v", err)
		}
		return filesys
	}
	filesys := setup_fs()

	mnt, err := fstestutil.MountedT(t, filesys, nil)
	if err != nil {
		t.Fatalf("Mount fail: %v\n", err)
	}
	defer mnt.Close()

	hello_path := path.Join(mnt.Dir, "second", "hello")
	f, err := os.Open(hello_path)
	if err != nil {
		t.Fatalf("hello open failed with %v", err)
	}
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("hello read failed with %v", err)
	}
	if string(buf) != GREETING {
		t.Errorf("hello read wrong content: %q", string(buf))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("hello close failed with %v", err)
	}
}

func TestJunkType(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()

	setup_fs := func() fs.FS {
		chunkStore := &mock.InMemory{}
		dir := setup_dir(t, chunkStore, []*wire.Dirent{
			&wire.Dirent{
				Name: "junk",
			},
		})
		filesys, err := newFS(chunkStore, dir)
		if err != nil {
			t.Fatalf("cannot serve snapshot as FUSE: %v", err)
		}
		return filesys
	}
	filesys := setup_fs()

	mnt, err := fstestutil.MountedT(t, filesys, nil)
	if err != nil {
		t.Fatalf("Mount fail: %v\n", err)
	}
	defer mnt.Close()

	junk_path := path.Join(mnt.Dir, "junk")
	_, err = os.Stat(junk_path)
	if err == nil {
		t.Fatalf("junk getattr must fail")
	} else if err.(*os.PathError).Err != syscall.EIO {
		t.Errorf("junk stat gave bad error: %v", err)
	}
}
