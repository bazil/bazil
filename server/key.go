package server

import (
	"crypto/rand"
	"fmt"

	"bazil.org/bazil/tokens"
	"github.com/agl/ed25519"
	"github.com/agl/ed25519/extra25519"
	"github.com/boltdb/bolt"
)

type CryptoKeys struct {
	Sign struct {
		Pub  *[ed25519.PublicKeySize]byte
		Priv *[ed25519.PrivateKeySize]byte
	}
	Box struct {
		Pub  *[32]byte
		Priv *[32]byte
	}
}

func extractEd25519Pubkey(priv *[ed25519.PrivateKeySize]byte) *[ed25519.PublicKeySize]byte {
	var pub [ed25519.PublicKeySize]byte
	copy(pub[:], priv[32:])
	return &pub
}

// loadOrGenerateKeys reads the master signing key from the global
// state in DB, (generating one if it's not already there), and
// generates the boxing keys based on it.
//
// This is meant to be called exactly once at startup time.
func loadOrGenerateKeys(db *bolt.DB) (*CryptoKeys, error) {
	var k CryptoKeys

	getKey := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketBazil))
		val := bucket.Get([]byte(tokens.GlobalStateKey))
		if val == nil {
			// have not generated a key yet
			return nil
		}
		if len(val) != ed25519.PrivateKeySize {
			return fmt.Errorf("master key wrong is the wrong size: length=%d", len(val))
		}
		var sigPriv [ed25519.PrivateKeySize]byte
		copy(sigPriv[:], val)
		k.Sign.Priv = &sigPriv
		return nil
	}
	if err := db.View(getKey); err != nil {
		return nil, err
	}

	if k.Sign.Priv == nil {
		// did not load keys from database
		var err error
		_, signPriv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		k.Sign.Priv = signPriv

		// save it for future runs
		putKey := func(tx *bolt.Tx) error {
			bucket := tx.Bucket([]byte(tokens.BucketBazil))
			return bucket.Put([]byte(tokens.GlobalStateKey), signPriv[:])
		}
		if err := db.Update(putKey); err != nil {
			return nil, err
		}
	}

	k.Sign.Pub = extractEd25519Pubkey(k.Sign.Priv)

	// generate other keys from it
	k.Box.Priv = &[32]byte{}
	extra25519.PrivateKeyToCurve25519(k.Box.Priv, k.Sign.Priv)
	k.Box.Pub = &[32]byte{}
	extra25519.PublicKeyToCurve25519(k.Box.Pub, k.Sign.Pub)
	return &k, nil
}
