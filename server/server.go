package server

import (
	"os"
	"path/filepath"

	"bazil.org/bazil/fs"
	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/tokens"
	"github.com/boltdb/bolt"
)

type App struct {
	DataDir string
	DB      *bolt.DB
}

func New(dataDir string) (app *App, err error) {
	err = os.Mkdir(dataDir, 0700)
	if err != nil && !os.IsExist(err) {
		return nil, err
	}

	kvpath := filepath.Join(dataDir, "chunks")
	err = kvfiles.Create(kvpath)
	if err != nil {
		return nil, err
	}

	dbpath := filepath.Join(dataDir, "bazil.bolt")
	db, err := bolt.Open(dbpath, 0600)
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
		if err := fs.Init(tx); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	app = &App{
		DataDir: dataDir,
		DB:      db,
	}
	return app, nil
}

func (app *App) Close() {
	app.DB.Close()
}
