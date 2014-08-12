package inodes_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"bazil.org/bazil/fs/inodes"
	"github.com/boltdb/bolt"
)

func TestAllocate(t *testing.T) {
	tmp, err := ioutil.TempFile("", "bazil-test-inodes-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	db, err := bolt.Open(tmp.Name(), 0666, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var inodeBucketName = []byte("inodetest")
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(inodeBucketName)
		return err
	})

	const firstInode = 1024
	for i := uint64(firstInode); i < firstInode+100; i++ {
		var got uint64
		f := func(tx *bolt.Tx) error {
			bucket := tx.Bucket(inodeBucketName)
			if bucket == nil {
				return fmt.Errorf("inode bucket missing in test: %q", inodeBucketName)
			}
			var err error
			got, err = inodes.Allocate(bucket)
			return err
		}
		if err := db.Update(f); err != nil {
			t.Error(err)
		}
		if g, e := got, i; g != e {
			t.Errorf("wrong inode allocated: %d != %d", g, e)
		}
	}
}

func TestAllocateMultiple(t *testing.T) {
	tmp, err := ioutil.TempFile("", "bazil-test-inodes-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		tmp.Close()
		os.Remove(tmp.Name())
	}()

	db, err := bolt.Open(tmp.Name(), 0666, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var inodeBucketName = []byte("inodetest")
	db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(inodeBucketName)
		return err
	})

	const firstInode = 1024
	for i := uint64(firstInode); i < firstInode+100; i += 2 {
		var gotOne, gotTwo uint64
		f := func(tx *bolt.Tx) error {
			bucket := tx.Bucket(inodeBucketName)
			if bucket == nil {
				return fmt.Errorf("inode bucket missing in test: %q", inodeBucketName)
			}
			var err error
			gotOne, err = inodes.Allocate(bucket)
			if err != nil {
				return err
			}
			gotTwo, err = inodes.Allocate(bucket)
			return err
		}
		if err := db.Update(f); err != nil {
			t.Error(err)
		}
		if g, e := gotOne, i; g != e {
			t.Errorf("wrong inode allocated: %d != %d", g, e)
		}
		if g, e := gotTwo, i+1; g != e {
			t.Errorf("wrong inode allocated: %d != %d", g, e)
		}
	}
}
