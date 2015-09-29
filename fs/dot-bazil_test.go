package fs_test

import (
	"fmt"
	"os"
	"path"
	"testing"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/util/tempdir"
)

func TestDotBazilContents(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()
	bazfstestutil.CreateVolume(t, app, "default")

	mnt := bazfstestutil.Mounted(t, app, "default")
	defer mnt.Close()

	p := path.Join(mnt.Dir, ".bazil")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat of .bazil failed: %v", err)
	}
	if g, e := fi.Mode(), os.ModeDir|0555; g != e {
		t.Errorf("wrong mode: %v != %v", g, e)
	}

	checkers := map[string]bazfstestutil.FileInfoCheck{
		"pending": func(fi os.FileInfo) error {
			if g, e := fi.Mode(), os.ModeDir|0500; g != e {
				return fmt.Errorf("wrong mode: %v != %v", g, e)
			}
			return nil
		},
	}
	if err := bazfstestutil.CheckDir(p, checkers); err != nil {
		t.Error(err)
	}
}
