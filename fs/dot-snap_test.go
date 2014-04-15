package fs_test

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/util/tempdir"
)

const GREETING = "hello, world\n"

func TestSnapRecord(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()
	bazfstestutil.CreateVolume(t, app, "default")

	mnt := bazfstestutil.Mounted(t, app, "default")
	defer mnt.Close()

	// write test data
	{
		sub := path.Join(mnt.Dir, "greetings")
		err := os.Mkdir(sub, 0755)
		if err != nil {
			t.Fatalf("cannot make directory: %v", err)
		}
		p := path.Join(sub, "hello")
		f, err := os.Create(p)
		if err != nil {
			t.Fatalf("cannot create hello: %v", err)
		}
		defer f.Close()
		_, err = f.Write([]byte(GREETING))
		if err != nil {
			t.Fatalf("cannot write to hello: %v", err)
		}
		err = f.Close()
		if err != nil {
			t.Fatalf("closing hello failed: %v", err)
		}
	}

	// make a snapshot
	{
		err := os.Mkdir(path.Join(mnt.Dir, ".snap", "mysnap"), 0755)
		if err != nil {
			t.Fatalf("snapshot failed: %v", err)
		}
	}

	// verify snapshot contents
	{
		data, err := ioutil.ReadFile(path.Join(mnt.Dir, ".snap", "mysnap", "greetings", "hello"))
		if err != nil {
			t.Fatalf("reading greeting failed: %v\n", err)
		}
		if g, e := string(data), GREETING; g != e {
			t.Errorf("wrong greeting: %q != %q", g, e)
		}
	}
}

func TestSnapList(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()
	bazfstestutil.CreateVolume(t, app, "default")

	mnt := bazfstestutil.Mounted(t, app, "default")
	defer mnt.Close()

	// make some snapshots
	{
		err := os.Mkdir(path.Join(mnt.Dir, ".snap", "snapone"), 0755)
		if err != nil {
			t.Fatalf("snapshot failed: %v", err)
		}
		err = os.Mkdir(path.Join(mnt.Dir, ".snap", "snaptwo"), 0755)
		if err != nil {
			t.Fatalf("snapshot failed: %v", err)
		}
		err = os.Mkdir(path.Join(mnt.Dir, ".snap", "alphabetical"), 0755)
		if err != nil {
			t.Fatalf("snapshot failed: %v", err)
		}
	}

	// list snapshots
	{
		fis, err := ioutil.ReadDir(path.Join(mnt.Dir, ".snap"))
		if err != nil {
			t.Fatalf("listing snapshots failed: %v\n", err)
		}
		for _, fi := range fis {
			if fi.Mode() != os.ModeDir|0555 {
				t.Errorf("snapshot has bad mode: %q is %#o", fi.Name(), fi.Mode())
			}
			// TODO fi.ModTime()
		}
		if g, e := len(fis), 3; g != e {
			t.Fatalf("wrong number of snapshots: %d != %d: %v", g, e, fis)
		}
		expect := []string{"alphabetical", "snapone", "snaptwo"}
		for i, fi := range fis {
			if g, e := fi.Name(), expect[i]; g != e {
				t.Errorf("wrong snapshot entry: %q != %q", g, e)
			}
		}
	}
}
