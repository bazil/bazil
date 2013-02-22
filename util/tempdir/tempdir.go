package tempdir

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

type Dir struct {
	Path string
	t    *testing.T
}

func (d Dir) Cleanup() {
	err := os.RemoveAll(d.Path)
	if err != nil {
		d.t.Errorf("tempdir cleanup failed: %v", err)
	}
}

// Check whether the given directory is empty. Marks test failed on
// problems, and on any files seen.
func (d Dir) CheckEmpty() {
	f, err := os.Open(d.Path)
	if err != nil {
		d.t.Errorf("Cannot open temp directory: %v", err)
		return
	}
	junk, err := f.Readdirnames(-1)
	if err != nil {
		d.t.Errorf("Cannot list temp directory: %v", err)
		return
	}
	if len(junk) != 0 {
		d.t.Errorf("Temp directory has unexpected junk: %v", junk)
		return
	}
}

func New(t *testing.T) Dir {
	// blatantly assuming we run under "go test"
	parent := path.Dir(os.Args[0])
	if path.Base(parent) != "_test" {
		t.Fatal("tempdir only works under 'go test'")
	}
	dir, err := ioutil.TempDir(parent, "temp-")
	if err != nil {
		t.Fatalf("cannot create temp directory: %v", err)
	}
	return Dir{Path: dir, t: t}
}
