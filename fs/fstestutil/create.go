package fstestutil

import (
	"path/filepath"
	"testing"

	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/fs"
	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/server"
	"bazil.org/fuse/fs/fstestutil"
)

func NewApp(t testing.TB, dataDir string) *server.App {
	app, err := server.New(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

// TODO this vs. bazil.org/fuse/fs/fstestutil#Mounted
func Mounted(t testing.TB, app *server.App) *fstestutil.Mount {
	// TODO this doesn't belong here
	kvpath := filepath.Join(app.DataDir, "chunks")
	kvstore, err := kvfiles.Open(kvpath)
	if err != nil {
		t.Fatal(err)
	}
	chunkStore := kvchunks.New(kvstore)

	filesys, err := fs.Open(app.DB, chunkStore)
	if err != nil {
		t.Fatalf("FS new fail: %v\n", err)
	}

	info, err := fstestutil.MountedT(t, filesys)
	if err != nil {
		t.Fatal(err)
	}

	return info
}
