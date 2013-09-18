package fs_test

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/util/tempdir"
)

func TestSimple(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	fi, err := os.Stat(mnt.Dir)
	if err != nil {
		t.Fatalf("root getattr failed with %v", err)
	}
	mode := fi.Mode()
	if (mode & os.ModeType) != os.ModeDir {
		t.Errorf("root is not a directory: %#v", fi)
	}
	if mode.Perm() != 0755 {
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

	dirf, err := os.Open(mnt.Dir)
	if err != nil {
		t.Fatalf("cannot open root dir: %v", err)
	}
	defer dirf.Close()
	names, err := dirf.Readdirnames(10)
	if err != nil && err != io.EOF {
		t.Fatalf("cannot list root dir: %v", err)
	}
	if len(names) > 0 {
		t.Errorf("unexpected content in root dir: %v", names)
	}
	err = dirf.Close()
	if err != nil {
		t.Fatalf("closing root dir failed: %v", err)
	}
}

func TestCreateFile(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}
	defer f.Close()
	GREETING := "hello, world\n"
	n, err := f.Write([]byte(GREETING))
	if err != nil {
		t.Fatalf("cannot write to hello: %v", err)
	}
	if n != len(GREETING) {
		t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	dirf, err := os.Open(mnt.Dir)
	if err != nil {
		t.Fatalf("cannot open root dir: %v", err)
	}
	defer dirf.Close()
	names, err := dirf.Readdirnames(10)
	if err != nil && err != io.EOF {
		t.Fatalf("cannot list root dir: %v", err)
	}
	if len(names) != 1 {
		t.Errorf("unexpected content in root dir: %v", names)
	}
	if names[0] != "hello" {
		t.Errorf("unexpected file in root dir: %q", names[0])
	}
}

func TestReadFile(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}
	defer f.Close()
	GREETING := "hello, world\n"
	n, err := f.Write([]byte(GREETING))
	if err != nil {
		t.Fatalf("cannot write to hello: %v", err)
	}
	if n != len(GREETING) {
		t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	f, err = os.Open(p)
	if err != nil {
		t.Fatalf("cannot open hello: %v", err)
	}
	defer f.Close()
	buf, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatalf("cannot read from hello: %v", err)
	}
	if string(buf) != GREETING {
		t.Fatalf("hello content is wrong: %q", buf)
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}
}

func TestAppend(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}
	defer f.Close()
	GREETING := "hello, world\n"
	n, err := f.Write([]byte(GREETING))
	if err != nil {
		t.Fatalf("cannot write to hello: %v", err)
	}
	if n != len(GREETING) {
		t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	f2, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Fatalf("cannot open hello again: %v", err)
	}
	defer f2.Close()
	GREETING2 := "more\n"
	n, err = f2.Write([]byte(GREETING2))
	if err != nil {
		t.Fatalf("cannot append to hello: %v", err)
	}
	if n != len(GREETING2) {
		t.Fatalf("bad length append to hello: %d != %d", n, len(GREETING2))
	}
	err = f2.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	f3, err := os.Open(p)
	if err != nil {
		t.Fatalf("cannot open hello: %v", err)
	}
	defer f3.Close()
	buf, err := ioutil.ReadAll(f3)
	if err != nil {
		t.Fatalf("cannot read from hello: %v", err)
	}
	if string(buf) != GREETING+GREETING2 {
		t.Fatalf("hello content is wrong: %q", buf)
	}
	err = f3.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}
}

func TestMkdir(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "sub")
	err := os.Mkdir(p, 0700)
	if err != nil {
		t.Fatalf("cannot mkdir sub: %v", err)
	}

	dirf, err := os.Open(mnt.Dir)
	if err != nil {
		t.Fatalf("cannot open root dir: %v", err)
	}
	defer dirf.Close()
	names, err := dirf.Readdirnames(10)
	if err != nil && err != io.EOF {
		t.Fatalf("cannot list root dir: %v", err)
	}
	if len(names) != 1 {
		t.Fatalf("unexpected content in root dir: %v", names)
	}
	if names[0] != "sub" {
		t.Errorf("unexpected file in root dir: %q", names[0])
	}
}

func TestStatFile(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}
	defer f.Close()
	GREETING := "hello, world\n"
	n, err := f.Write([]byte(GREETING))
	if err != nil {
		t.Fatalf("cannot write to hello: %v", err)
	}
	if n != len(GREETING) {
		t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("cannot stat hello: %v", err)
	}
	mode := fi.Mode()
	if (mode & os.ModeType) != 0 {
		t.Errorf("hello is not a file: %#v", fi)
	}
	if mode.Perm() != 0644 {
		t.Errorf("file has weird access mode: %v", mode.Perm())
	}
	switch stat := fi.Sys().(type) {
	case *syscall.Stat_t:
		if stat.Nlink != 1 {
			t.Errorf("file has wrong link count: %v", stat.Nlink)
		}
		if stat.Uid != uint32(syscall.Getuid()) {
			t.Errorf("file has wrong uid: %d", stat.Uid)
		}
		if stat.Gid != uint32(syscall.Getgid()) {
			t.Errorf("file has wrong gid: %d", stat.Gid)
		}
		if stat.Gid != uint32(syscall.Getgid()) {
			t.Errorf("file has wrong gid: %d", stat.Gid)
		}
	}
	if fi.Size() != int64(len(GREETING)) {
		t.Errorf("file has wrong size: %d != %d", fi.Size(), len(GREETING))
	}
}

func TestPersistentMkdir(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	func() {
		mnt := bazfstestutil.Mounted(t, app)
		defer mnt.Close()

		p := path.Join(mnt.Dir, "subdir")
		err := os.Mkdir(p, 0755)
		if err != nil {
			t.Fatalf("cannot create subdir: %v", err)
		}
	}()

	t.Logf("Unmounted to check persistency")

	func() {
		mnt := bazfstestutil.Mounted(t, app)
		defer mnt.Close()

		dirf, err := os.Open(mnt.Dir)
		if err != nil {
			t.Fatalf("cannot open root dir: %v", err)
		}
		defer dirf.Close()
		names, err := dirf.Readdirnames(10)
		if err != nil && err != io.EOF {
			t.Fatalf("cannot list root dir: %v", err)
		}
		if len(names) != 1 {
			t.Errorf("unexpected content in root dir: %v", names)
		}
		if len(names) > 0 && names[0] != "subdir" {
			t.Errorf("unexpected file in root dir: %q", names[0])
		}

		p := path.Join(mnt.Dir, "subdir")
		fi, err := os.Stat(p)
		if err != nil {
			t.Fatalf("cannot stat subdir: %v", err)
		}
		if !fi.IsDir() {
			t.Fatalf("subdir is not a directory: %v", fi)
		}
	}()
}

func TestPersistentCreateFile(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	GREETING := "hello, world\n"

	func() {
		mnt := bazfstestutil.Mounted(t, app)
		defer mnt.Close()

		p := path.Join(mnt.Dir, "hello")
		f, err := os.Create(p)
		if err != nil {
			t.Fatalf("cannot create hello: %v", err)
		}
		defer f.Close()
		n, err := f.Write([]byte(GREETING))
		if err != nil {
			t.Fatalf("cannot write to hello: %v", err)
		}
		if n != len(GREETING) {
			t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
		}
		err = f.Close()
		if err != nil {
			t.Fatalf("closing hello failed: %v", err)
		}
	}()

	t.Logf("Unmounted to check persistency")

	func() {
		mnt := bazfstestutil.Mounted(t, app)
		defer mnt.Close()

		dirf, err := os.Open(mnt.Dir)
		if err != nil {
			t.Fatalf("cannot open root dir: %v", err)
		}
		defer dirf.Close()
		names, err := dirf.Readdirnames(10)
		if err != nil && err != io.EOF {
			t.Fatalf("cannot list root dir: %v", err)
		}
		if len(names) != 1 {
			t.Errorf("unexpected content in root dir: %v", names)
		}
		if len(names) > 0 && names[0] != "hello" {
			t.Errorf("unexpected file in root dir: %q", names[0])
		}

		p := path.Join(mnt.Dir, "hello")
		f, err := os.Open(p)
		if err != nil {
			t.Fatalf("cannot open hello: %v", err)
		}
		defer f.Close()
		buf, err := ioutil.ReadAll(f)
		if err != nil {
			t.Fatalf("cannot read from hello: %v", err)
		}
		if string(buf) != GREETING {
			t.Fatalf("hello content is wrong: %q", buf)
		}
		err = f.Close()
		if err != nil {
			t.Fatalf("closing hello failed: %v", err)
		}
	}()
}

func TestRemoveFile(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	GREETING := "hello, world\n"
	err := ioutil.WriteFile(p, []byte(GREETING), 0644)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}

	err = os.Remove(p)
	if err != nil {
		t.Fatalf("cannot delete hello: %v", err)
	}

	dirf, err := os.Open(mnt.Dir)
	if err != nil {
		t.Fatalf("cannot open root dir: %v", err)
	}
	defer dirf.Close()
	names, err := dirf.Readdirnames(10)
	if err != nil && err != io.EOF {
		t.Fatalf("cannot list root dir: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("unexpected content in root dir: %v", names)
	}
}

func TestRemoveNonexistent(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "does-not-exist")
	err := os.Remove(p)
	if err == nil {
		t.Fatalf("deleting non-existent file should have failed")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("deleting non-existent file gave wrong error: %v", err)
	}
}

func TestRemoveFileWhileOpen(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}
	defer f.Close()

	err = os.Remove(p)
	if err != nil {
		t.Fatalf("cannot delete hello: %v", err)
	}

	// this must not resurrect a deleted file
	GREETING := "hello, world\n"
	n, err := f.Write([]byte(GREETING))
	if err != nil {
		t.Fatalf("cannot write to hello: %v", err)
	}
	if n != len(GREETING) {
		t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	dirf, err := os.Open(mnt.Dir)
	if err != nil {
		t.Fatalf("cannot open root dir: %v", err)
	}
	defer dirf.Close()
	names, err := dirf.Readdirnames(10)
	if err != nil && err != io.EOF {
		t.Fatalf("cannot list root dir: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("unexpected content in root dir: %v", names)
	}
}

func TestTruncate(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(t, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "hello")
	f, err := os.Create(p)
	if err != nil {
		t.Fatalf("cannot create hello: %v", err)
	}
	defer f.Close()
	GREETING := "hello, world\n"
	n, err := f.Write([]byte(GREETING))
	if err != nil {
		t.Fatalf("cannot write to hello: %v", err)
	}
	if n != len(GREETING) {
		t.Fatalf("bad length write to hello: %d != %d", n, len(GREETING))
	}
	err = f.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}

	err = os.Truncate(p, 3)
	if err != nil {
		t.Fatalf("truncate failed: %v", err)
	}

	f2, err := os.Open(p)
	if err != nil {
		t.Fatalf("cannot open hello: %v", err)
	}
	defer f2.Close()
	buf, err := ioutil.ReadAll(f2)
	if err != nil {
		t.Fatalf("cannot read from hello: %v", err)
	}
	if g, e := string(buf), GREETING[:3]; g != e {
		t.Fatalf("hello content is wrong: %q != %q", g, e)
	}
	err = f2.Close()
	if err != nil {
		t.Fatalf("closing hello failed: %v", err)
	}
}
