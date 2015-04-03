package server

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs"
	"bazil.org/bazil/fs/wire"
	"bazil.org/bazil/kv"
	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/kv/kvmulti"
	"bazil.org/bazil/kv/kvpeer"
	"bazil.org/bazil/kv/untrusted"
	"bazil.org/bazil/peer"
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
	DB       *db.DB
	mounts   struct {
		sync.Mutex
		open map[fs.VolumeID]*mountState
	}
	Keys *CryptoKeys
	tls  struct {
		config atomic.Value
		gen    sync.Mutex
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
	database, err := db.Open(dbpath, 0600, nil)
	if err != nil {
		return nil, err
	}

	err = database.DB.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketBazil)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketVolume)); err != nil {
			return err
		}
		if _, err := tx.CreateBucketIfNotExists([]byte(tokens.BucketVolName)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		database.Close()
		return nil, err
	}

	keys, err := loadOrGenerateKeys(database.DB)
	if err != nil {
		return nil, err
	}

	app = &App{
		DataDir:  dataDir,
		lockFile: lockFile,
		DB:       database,
		Keys:     keys,
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

func (app *App) openKV(tx *db.Tx, conf []*wire.VolumeStorage) (kv.KV, error) {
	var kvstores []kv.KV

	for _, storage := range conf {
		s, err := app.openStorage(storage.Backend)
		if err != nil {
			return nil, err
		}

		secret, err := tx.SharingKeys().Get(storage.SharingKeyName)
		if err != nil {
			return nil, fmt.Errorf("getting sharing key %q: %v", storage.SharingKeyName, err)
		}
		s = untrusted.New(s, secret)
		kvstores = append(kvstores, s)
	}

	return kvmulti.New(kvstores...), nil
}

func (app *App) openStorage(backend string) (kv.KV, error) {
	switch backend {
	case "local":
		kvpath := filepath.Join(app.DataDir, "chunks")
		return kvfiles.Open(kvpath)
	}
	if backend != "" && backend[0] == '/' {
		return kvfiles.Open(backend)
	}
	idx := strings.IndexByte(backend, ':')
	if idx != -1 {
		scheme, rest := backend[:idx], backend[idx+1:]
		switch scheme {
		case "peerkey":
			var key peer.PublicKey
			if err := key.Set(rest); err != nil {
				return nil, err
			}
			p, err := app.DialPeer(&key)
			if err != nil {
				return nil, err
			}
			// TODO Close
			return kvpeer.Open(p)
		}
	}
	return nil, errors.New("unknown storage backend")
}

func (app *App) ValidateKV(backend string) error {
	// TODO opening a KV just to validate the input string would be a
	// bad idea if a backend was costly to open; then again, current
	// API doesn't support a Close anyway.
	_, err := app.openStorage(backend)
	return err
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
	err := app.DB.View(func(tx *db.Tx) error {
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

		kvstore, err := app.openKV(tx, volConf.Storage)
		if err != nil {
			return err
		}

		chunkStore := kvchunks.New(kvstore)
		vol, err = fs.Open(app.DB.DB, chunkStore, volumeID)
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
