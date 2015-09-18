package flagx

import (
	"encoding/hex"
	"errors"
	"flag"

	"bazil.org/bazil/cas"
)

// KeyParam is a wrapper for cas.Key that is compatible with the
// flag.Value interface, without compromising the immutability promise
// of Key.
type KeyParam struct {
	key cas.Key
}

var _ flag.Value = (*KeyParam)(nil)

// String returns a hex representation of the key.
//
// See flag.Value.String and cas.Key.String.
func (kp KeyParam) String() string {
	return kp.key.String()
}

// Set updates the contents of the key based on the given hex-encoded
// string.
//
// See flag.Value.Set.
func (kp *KeyParam) Set(s string) error {
	buf, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	if len(buf) != cas.KeySize {
		return &cas.BadKeySizeError{Key: buf}
	}
	k := cas.NewKey(buf)
	if k == cas.Invalid {
		return errors.New("bad key format")
	}
	kp.key = k
	return nil
}

// Key returns a copy of the current value of the KeyParam.
//
// As usual with cas.Key, the returned value is promised to be
// immutable.
func (kp *KeyParam) Key() cas.Key {
	return kp.key
}
