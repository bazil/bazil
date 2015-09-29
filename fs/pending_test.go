package fs_test

import (
	"os"
	"path"
	"testing"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/util/tempdir"
)

func TestPendingListEmpty(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()
	bazfstestutil.CreateVolume(t, app, "default")

	mnt := bazfstestutil.Mounted(t, app, "default")
	defer mnt.Close()

	p := path.Join(mnt.Dir, ".bazil", "pending")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat of pending dir failed: %v", err)
	}
	if g, e := fi.Mode(), os.ModeDir|0500; g != e {
		t.Errorf("wrong mode: %v != %v", g, e)
	}

	if err := bazfstestutil.CheckDir(p, nil); err != nil {
		t.Error(err)
	}
}
