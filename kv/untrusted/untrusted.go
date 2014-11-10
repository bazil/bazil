package untrusted

import (
	"fmt"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/kv"
	"bazil.org/bazil/tokens"
	"github.com/codahale/blake2"
	"golang.org/x/crypto/nacl/secretbox"
)

type Convergent struct {
	// not a CAS because key derives from the plaintext version, but
	// used like a Fixed Content Storage (FCS)
	untrusted kv.KV
	secret    *[32]byte
}

var _ = kv.KV(&Convergent{})

var personalizeKey = []byte(tokens.Blake2bPersonalizationConvergentKey)

func (s *Convergent) computeBoxedKey(key []byte) []byte {
	conf := blake2.Config{
		Size:     cas.KeySize,
		Key:      s.secret[:],
		Personal: personalizeKey,
	}
	h := blake2.New(&conf)
	// hash.Hash docs say it never fails
	_, _ = h.Write(key)
	return h.Sum(nil)
}

const nonceSize = 24

var personalizeNonce = []byte(tokens.Blake2bPersonalizationConvergentNonce)

// Nonce summarizes key, type and level so mismatch of e.g. type can
// be detected.
func (s *Convergent) makeNonce(key []byte) *[nonceSize]byte {
	conf := blake2.Config{
		Size:     nonceSize,
		Personal: personalizeNonce,
	}
	h := blake2.New(&conf)
	// hash.Hash docs say it never fails
	_, _ = h.Write(key)

	var ret [nonceSize]byte
	h.Sum(ret[:0])
	return &ret
}

func (s *Convergent) Get(key []byte) ([]byte, error) {
	boxedkey := s.computeBoxedKey(key)
	box, err := s.untrusted.Get(boxedkey)
	if err != nil {
		return nil, err
	}

	nonce := s.makeNonce(key)
	plain, ok := secretbox.Open(nil, box, nonce, s.secret)
	if !ok {
		return nil, Corrupt{Key: key}
	}
	return plain, nil
}

func (s *Convergent) Put(key []byte, value []byte) error {
	nonce := s.makeNonce(key)
	box := secretbox.Seal(nil, value, nonce, s.secret)

	boxedkey := s.computeBoxedKey(key)
	err := s.untrusted.Put(boxedkey, box)
	return err
}

func New(store kv.KV, secret *[32]byte) *Convergent {
	return &Convergent{
		untrusted: store,
		secret:    secret,
	}
}

type Corrupt struct {
	Key []byte
}

func (c Corrupt) Error() string {
	return fmt.Sprintf("corrupt encrypted chunk: %x", c.Key)
}

var _ = error(Corrupt{})
