package db_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"

	"bazil.org/bazil/db"
)

type TestDB struct {
	*db.DB
}

func NewTestDB(t testing.TB) *TestDB {
	f, err := ioutil.TempFile("", "bazil-test-db-")
	if err != nil {
		t.Fatalf("cannot create temp file: %v", err)
	}
	path := f.Name()
	f.Close()

	db, err := db.Open(path, 0600, &bolt.Options{Timeout: 1 * time.Nanosecond})
	if err != nil {
		t.Fatalf("db open: %v", err)
	}
	return &TestDB{db}
}

func (db *TestDB) Close() {
	defer os.Remove(db.Path())
	db.DB.Close()
}
