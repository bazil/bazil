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

	mnt := bazfstestutil.Mounted(t, app)
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
