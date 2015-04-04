package fstestutil

import (
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"bazil.org/bazil/db"
	"bazil.org/bazil/server"
	"bazil.org/fuse"
)

func NewApp(t testing.TB, dataDir string) *server.App {
	app, err := server.New(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	return app
}

func CreateVolume(t testing.TB, app *server.App, volumeName string) {
	createVolume := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Get("default")
		if err != nil {
			return err
		}
		if _, err := tx.Volumes().Create(volumeName, "local", sharingKey); err != nil {
			return err
		}
		return nil
	}
	if err := app.DB.Update(createVolume); err != nil {
		t.Fatal(err)
	}
}

type Mount struct {
	// Dir is the temporary directory where the filesystem is mounted.
	Dir  string
	Info *server.MountInfo

	app    *server.App
	closed bool
}

// Close unmounts the filesystem and waits for fs.Serve to return.
//
// TODO not true: Any returned error will be stored in Err.
//
//  It is safe to call Close multiple times.
func (mnt *Mount) Close() {
	if mnt.closed {
		return
	}
	mnt.closed = true
	for tries := 0; tries < 1000; tries++ {
		err := fuse.Unmount(mnt.Dir)
		if err != nil {
			// TODO do more than log?
			// TODO use lazy unmount where available?
			log.Printf("unmount error: %v", err)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		break
	}
	mnt.app.WaitForUnmount(&mnt.Info.VolumeID)
	os.Remove(mnt.Dir)
}

// TODO this vs. bazil.org/fuse/fs/fstestutil#Mounted
func Mounted(t testing.TB, app *server.App, volumeName string) *Mount {
	mountpoint, err := ioutil.TempDir("", "bazil-test-")
	if err != nil {
		t.Fatal(err)
	}

	// TODO make it log debug if `go test ./fs -fuse.debug`
	info, err := app.Mount(volumeName, mountpoint)
	if err != nil {
		t.Fatal(err)
	}

	mnt := &Mount{
		Dir:  mountpoint,
		Info: info,
		app:  app,
	}
	return mnt
}
