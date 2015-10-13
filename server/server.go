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
)

type App struct {
	DataDir  string
	lockFile *os.File
	DB       *db.DB
	debug    func(data interface{})
	volumes  struct {
		sync.Mutex
		// This Broadcasts whenever open volumes, or their mounted
		// state, changes.
		sync.Cond
		open map[db.VolumeID]*VolumeRef
	}
	Keys *CryptoKeys
	tls  struct {
		config atomic.Value
		gen    sync.Mutex
	}
}

func New(dataDir string, options ...AppOption) (app *App, err error) {
	config := &appConfig{}
	for _, option := range options {
		if err := option(config); err != nil {
			return nil, err
		}
	}

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
		debug:    config.debug,
		Keys:     keys,
	}
	app.volumes.Cond.L = &app.volumes.Mutex
	app.volumes.open = make(map[db.VolumeID]*VolumeRef)
	return app, nil
}

func (app *App) Close() {
	// Wait for VolumeRefs to go away, to detect refcounting bugs.
	app.volumes.Lock()
	for len(app.volumes.open) > 0 {
		app.volumes.Wait()
	}
	app.volumes.Unlock()

	app.DB.Close()
	app.lockFile.Close()
}

func (app *App) Debug(msg interface{}) {
	if app.debug == nil {
		return
	}
	app.debug(msg)
}

func (app *App) GetVolume(id *db.VolumeID) (*VolumeRef, error) {
	app.volumes.Lock()
	defer app.volumes.Unlock()

	ref, found := app.volumes.open[*id]
	if !found {
		open := func(tx *db.Tx) error {
			vol, err := app.openVolume(tx, id)
			if err != nil {
				return err
			}
			ref = &VolumeRef{
				app:   app,
				volID: *id,
				fs:    vol,
			}
			return nil
		}
		if err := app.DB.View(open); err != nil {
			return nil, err
		}
		app.volumes.open[*id] = ref
		app.volumes.Broadcast()
	}
	ref.refs++
	return ref, nil
}

func (app *App) GetVolumeByName(name string) (*VolumeRef, error) {
	var volID db.VolumeID
	find := func(tx *db.Tx) error {
		vol, err := tx.Volumes().GetByName(name)
		if err != nil {
			return err
		}
		vol.VolumeID(&volID)
		return nil
	}
	if err := app.DB.View(find); err != nil {
		return nil, err
	}
	return app.GetVolume(&volID)
}

// caller must hold App.volumes.Mutex
func (app *App) openVolume(tx *db.Tx, id *db.VolumeID) (*fs.Volume, error) {
	v, err := tx.Volumes().GetByVolumeID(id)
	if err != nil {
		return nil, err
	}
	kvstore, err := app.OpenKV(tx, v.Storage())
	if err != nil {
		return nil, err
	}

	chunkStore := kvchunks.New(kvstore)
	vol, err := fs.Open(app.DB, chunkStore, id, (*peer.PublicKey)(app.Keys.Sign.Pub))
	if err != nil {
		return nil, err
	}
	return vol, nil
}

func (app *App) OpenKV(tx *db.Tx, storage *db.VolumeStorage) (kv.KV, error) {
	var kvstores []kv.KV

	c := storage.Cursor()
	for item := c.First(); item != nil; item = c.Next() {
		backend, err := item.Backend()
		if err != nil {
			return nil, err
		}
		s, err := app.openStorage(backend)
		if err != nil {
			return nil, err
		}

		sharingKeyName, err := item.SharingKeyName()
		if err != nil {
			return nil, err
		}
		sharingKey, err := tx.SharingKeys().Get(sharingKeyName)
		if err != nil {
			return nil, fmt.Errorf("getting sharing key %q: %v", sharingKeyName, err)
		}
		var secret [32]byte
		sharingKey.Secret(&secret)
		s = untrusted.New(s, &secret)

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

type VolumeRef struct {
	app   *App
	volID db.VolumeID
	fs    *fs.Volume

	// fields protected by App.volumes.Mutex

	refs    uint32
	mounted bool
	conn    *fuse.Conn
}

func (ref *VolumeRef) Close() {
	ref.app.volumes.Lock()
	defer ref.app.volumes.Unlock()

	ref.refs--
	if ref.refs == 0 {
		delete(ref.app.volumes.open, ref.volID)
		ref.app.volumes.Broadcast()
	}
}

// FS returns the underlying filesystem implementation.
//
// Caller must keep a reference to VolumeRef for the duration
func (ref *VolumeRef) FS() *fs.Volume {
	return ref.fs
}

// Protocol returns the underlying FUSE protocol version.
//
// Caller must keep a reference to VolumeRef for the duration
func (ref *VolumeRef) Protocol() (*fuse.Protocol, error) {
	ref.app.volumes.Lock()
	defer ref.app.volumes.Unlock()
	if !ref.mounted {
		return nil, errors.New("not mounted")
	}
	p := ref.conn.Protocol()
	return &p, nil
}

// Mount makes the contents of the volume visible at the given
// mountpoint. If Mount returns with a nil error, the mount has
// occurred.
func (ref *VolumeRef) Mount(mountpoint string) error {
	ref.app.volumes.Lock()
	defer ref.app.volumes.Unlock()

	if ref.mounted {
		return errors.New("volume already mounted")
	}

	conn, err := fuse.Mount(mountpoint,
		fuse.MaxReadahead(32*1024*1024),
		fuse.AsyncRead(),
	)
	if err != nil {
		return fmt.Errorf("mount fail: %v", err)
	}

	srv := fusefs.New(conn, &fusefs.Config{
		Debug: ref.debug,
	})
	serveErr := make(chan error, 1)
	go func() {
		defer func() {
			// remove map entry on unmount or failed mount
			ref.app.volumes.Lock()
			ref.mounted = false
			ref.conn = nil
			ref.app.volumes.Unlock()
			ref.app.volumes.Broadcast()
			ref.Close()
		}()
		defer conn.Close()
		ref.fs.SetFUSE(srv)
		defer func() {
			ref.fs.SetFUSE(nil)
		}()
		serveErr <- srv.Serve(ref.fs)
	}()

	select {
	case <-conn.Ready:
		if err := conn.MountError; err != nil {
			return fmt.Errorf("mount fail (delayed): %v", err)
		}
		ref.refs++
		ref.mounted = true
		ref.conn = conn
		ref.app.volumes.Broadcast()
		return nil
	case err := <-serveErr:
		// Serve quit early
		if err != nil {
			return fmt.Errorf("filesystem failure: %v", err)
		}
		return errors.New("Serve exited early")
	}
}

type fuseDebug struct {
	VolumeID db.VolumeID
	Msg      interface{}
}

func (f *fuseDebug) String() string {
	// short prefix should be enough to identify which volume it is
	// when eyeballing logs
	vol := f.VolumeID.String()[:4]
	return fmt.Sprintf("%s %v", vol, f.Msg)
}

func (ref *VolumeRef) debug(msg interface{}) {
	ref.app.Debug(&fuseDebug{
		VolumeID: ref.volID,
		Msg:      msg,
	})
}

var ErrNotMounted = errors.New("not currently mounted")

func (ref *VolumeRef) WaitForUnmount() error {
	ref.app.volumes.Lock()
	defer ref.app.volumes.Unlock()
	if !ref.mounted {
		return ErrNotMounted
	}
	for ref.mounted {
		ref.app.volumes.Wait()
	}
	return nil
}
