package db

import (
	"crypto/rand"
	"errors"

	"bazil.org/bazil/tokens"
	"github.com/boltdb/bolt"
)

var (
	ErrSharingKeyNotFound = errors.New("sharing key not found")
	ErrSharingKeyExist    = errors.New("sharing key exists already")
)

var (
	bucketSharing = []byte(tokens.BucketSharing)
)

func (tx *Tx) initSharingKeys() error {
	if bucket := tx.Bucket(bucketSharing); bucket != nil {
		// All done; be careful to not recreate "default" key, if
		// removed.
		return nil
	}

	if _, err := tx.CreateBucketIfNotExists(bucketSharing); err != nil {
		return err
	}
	b := tx.SharingKeys()

	// Create the default sharing secret.
	const defaultKey = "default"
	var secret [32]byte
	if _, err := rand.Read(secret[:]); err != nil {
		return err
	}
	if err := b.Add(defaultKey, &secret); err != nil {
		return err
	}

	return nil
}

func (tx *Tx) SharingKeys() *SharingKeys {
	b := tx.Bucket(bucketSharing)
	return &SharingKeys{b}
}

type SharingKeys struct {
	b *bolt.Bucket
}

// Get a sharing key.
//
// If the sharing key name is not found, returns
// ErrSharingKeyNotFound.
//
// Returned key is valid after the transaction.
func (b *SharingKeys) Get(name string) (key *[32]byte, err error) {
	v := b.b.Get([]byte(name))
	if v == nil {
		return nil, ErrSharingKeyNotFound
	}
	var secret [32]byte
	copy(secret[:], v)
	return &secret, nil
}

// Add a sharing key.
//
// If a sharing key by that name already exists, returns
// ErrSharingKeyExist.
func (b *SharingKeys) Add(name string, key *[32]byte) error {
	if v := b.b.Get([]byte(name)); v != nil {
		return ErrSharingKeyExist
	}
	return b.b.Put([]byte(name), key[:])
}
