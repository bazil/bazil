package db

import (
	"crypto/rand"
	"errors"

	"bazil.org/bazil/tokens"
	"github.com/boltdb/bolt"
)

var (
	ErrSharingKeyNameInvalid = errors.New("invalid sharing key name")
	ErrSharingKeyNotFound    = errors.New("sharing key not found")
	ErrSharingKeyExist       = errors.New("sharing key exists already")
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
	if _, err := b.Add(defaultKey, &secret); err != nil {
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
func (b *SharingKeys) Get(name string) (*SharingKey, error) {
	n := []byte(name)
	v := b.b.Get(n)
	if v == nil {
		return nil, ErrSharingKeyNotFound
	}
	s := &SharingKey{
		b:      b,
		name:   n,
		secret: v,
	}
	return s, nil
}

// Add a sharing key.
//
// If name is invalid, returns ErrSharingKeyNameInvalid.
//
// If a sharing key by that name already exists, returns
// ErrSharingKeyExist.
func (b *SharingKeys) Add(name string, key *[32]byte) (*SharingKey, error) {
	if name == "" {
		return nil, ErrSharingKeyNameInvalid
	}
	n := []byte(name)
	if v := b.b.Get(n); v != nil {
		return nil, ErrSharingKeyExist
	}
	if err := b.b.Put([]byte(name), key[:]); err != nil {
		return nil, err
	}
	s := &SharingKey{
		b:      b,
		name:   n,
		secret: key[:],
	}
	return s, nil
}

type SharingKey struct {
	b      *SharingKeys
	name   []byte
	secret []byte
}

// Name returns the name of the sharing key.
//
// Returned value is valid after the transaction.
func (s *SharingKey) Name() string {
	return string(s.name)
}

// Secret copies the secret key to out.
//
// out is valid after the transaction.
func (s *SharingKey) Secret(out *[32]byte) {
	copy(out[:], s.secret)
}
