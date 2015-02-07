package server

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/fs"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/kv"
	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/kv/kvmulti"
	"bazil.org/bazil/kv/untrusted"
	"bazil.org/bazil/tokens"
	"bazil.org/fuse"
	fusefs "bazil.org/fuse/fs"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
)

type mountState struct {
	// closed after the serve loop has exited
	unmounted chan struct{}
}

type App struct {
	DataDir  string
	lockFile *os.File
	DB       *bolt.DB
	mounts   struct {
		sync.Mutex
		open map[fs.VolumeID]*mountState
	}
}

func New(dataDir string) (app *App, err error) {
	err = os.Mkdir(dataDir, 0700)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	lockPath := filepath.Join(dataDir, "lock")
	lockFile, err := lock(lockPath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			// if we're reporting an error, also unlock
			_ = lockFile.Close()
		}
	}()

	kvpath := filepath.Join(dataDir, "chunks")
	err = kvfiles.Create(kvpath)
	if err != nil {
		return nil, err
	}

	dbpath := filepath.Join(dataDir, "bazil.bolt")
	db, err := bolt.Open(dbpath, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = db.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketBazil)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketVolume)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketVolName)); err != nil {
			return err
		}
		bucket, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketSharing))
		if err != nil {
			return err
		}
		// Create the default sharing secret.
		var defaultKey = []byte("default")
		if bucket.Get(defaultKey) == nil {
			var secret [32]byte
			if _, err := rand.Read(secret[:]); err != nil {
				return err
			}
			if err := bucket.Put(defaultKey, secret[:]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	app = &App{
		DataDir:  dataDir,
		lockFile: lockFile,
		DB:       db,
	}
	app.mounts.open = make(map[fs.VolumeID]*mountState)
	return app, nil
}

func (app *App) Close() {
	app.DB.Close()
	app.lockFile.Close()
}

// TODO this function smells
func (app *App) serveMount(vol *fs.Volume, id *fs.VolumeID, mountpoint string) error {
	conn, err := fuse.Mount(mountpoint)
	if err != nil {
		// remove map entry if the mount never took place
		app.mounts.Lock()
		delete(app.mounts.open, *id)
		app.mounts.Unlock()
		return fmt.Errorf("mount fail: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		defer func() {
			// remove map entry on unmount or failed mount
			app.mounts.Lock()
			delete(app.mounts.open, *id)
			app.mounts.Unlock()
		}()
		defer conn.Close()
		serveErr <- fusefs.Serve(conn, vol)
	}()

	select {
	case <-conn.Ready:
		if conn.MountError != nil {
			return fmt.Errorf("mount fail (delayed): %v", err)
		}
		return nil
	case err = <-serveErr:
		// Serve quit early
		if err != nil {
			return fmt.Errorf("filesystem failure: %v", err)
		}
		return errors.New("Serve exited early")
	}
}

type MountInfo struct {
	VolumeID fs.VolumeID
}

func (app *App) openKV(conf *wire.KV) (kv.KV, error) {
	var kvstores []kv.KV

	if conf.Local != nil {
		kvpath := filepath.Join(app.DataDir, "chunks")
		var s kv.KV
		var err error
		s, err = kvfiles.Open(kvpath)
		if err != nil {
			return nil, err
		}
		if conf.Local.Secret != nil {
			var secret [32]byte
			copy(secret[:], conf.Local.Secret)
			s = untrusted.New(s, &secret)
		}
		kvstores = append(kvstores, s)
	}

	for _, ext := range conf.External {
		var s kv.KV
		var err error
		s, err = kvfiles.Open(ext.Path)
		if err != nil {
			return nil, err
		}
		if ext.Secret != nil {
			var secret [32]byte
			copy(secret[:], ext.Secret)
			s = untrusted.New(s, &secret)
		}
		kvstores = append(kvstores, s)
	}

	return kvmulti.New(kvstores...), nil
}

// Mount makes the contents of the volume visible at the given
// mountpoint. If Mount returns with a nil error, the mount has
// occurred.
func (app *App) Mount(volumeName string, mountpoint string) (*MountInfo, error) {
	// TODO obey `bazil -debug server run`

	var vol *fs.Volume
	var volumeID *fs.VolumeID
	var ready = make(chan error, 1)
	app.mounts.Lock()
	err := app.DB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketVolName))
		val := bucket.Get([]byte(volumeName))
		if val == nil {
			return errors.New("volume not found")
		}
		var volConf wire.VolumeConfig
		if err := proto.Unmarshal(val, &volConf); err != nil {
			return err
		}
		var err error
		volumeID, err = fs.NewVolumeID(volConf.VolumeID)
		if err != nil {
			return err
		}
		if _, ok := app.mounts.open[*volumeID]; ok {
			return errors.New("volume already mounted")
		}

		kvstore, err := app.openKV(volConf.Storage)
		if err != nil {
			return err
		}

		chunkStore := kvchunks.New(kvstore)
		vol, err = fs.Open(app.DB, chunkStore, volumeID)
		if err != nil {
			return err
		}
		mnt := &mountState{
			unmounted: make(chan struct{}),
		}
		go func() {
			defer close(mnt.unmounted)
			ready <- app.serveMount(vol, volumeID, mountpoint)
		}()
		app.mounts.open[*volumeID] = mnt
		return nil
	})
	app.mounts.Unlock()
	if err != nil {
		return nil, err
	}
	err = <-ready
	if err != nil {
		return nil, err
	}
	info := &MountInfo{
		VolumeID: *volumeID,
	}
	return info, nil
}

var ErrNotMounted = errors.New("not currently mounted")

func (app *App) WaitForUnmount(volumeID *fs.VolumeID) error {
	app.mounts.Lock()
	// we hold onto mnt after releasing the lock, but it's safe in
	// this case; gc keeps it pinned, and we don't look at mutable
	// data
	mnt, ok := app.mounts.open[*volumeID]
	app.mounts.Unlock()
	if !ok {
		return ErrNotMounted
	}
	<-mnt.unmounted
	return nil
}
