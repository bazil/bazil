package cas

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"

	"bazil.org/bazil/pb"

	"github.com/gogo/protobuf/proto"
)

// Size of the CAS keys in bytes.
const KeySize = 64

// Keys
//   0000..00xxxxxxxxxxxxxxxxxx (72-bit space) are special. This includes:
//
//   - 0000..000000000000000000: Empty key. Valid input.
//   - 0000..FE0BADBADBADBADBAD: Invalid key. Invalid input. All invalid inputs become this.
//   - 0000..FFxxxxxxxxxxxxxxxx: Private use. Valid input for NewKeyPrivate, not valid for NewKey.
//   - rest: reserved for future use. Not valid input.
var specialPrefix = make([]byte, KeySize-9)

// BadKeySizeError is passed to panic if NewKey is called with input
// that is not KeySize long.
type BadKeySizeError struct {
	Key []byte
}

var _ = error(&BadKeySizeError{})

func (b *BadKeySizeError) Error() string {
	return fmt.Sprintf("Key is bad length %d: %x", len(b.Key), b.Key)
}

// A Key that identifies data stored in the CAS. Keys are immutable.
type Key struct {
	object [KeySize]byte
}

// String returns a hex encoding of the key.
func (k Key) String() string {
	return hex.EncodeToString(k.object[:])
}

// Bytes returns a byte slice with the byte content of the key.
func (k *Key) Bytes() []byte {
	buf := make([]byte, KeySize)
	copy(buf, k.object[:])
	return buf
}

// IsSpecial returns true if the key is special.
// Special keys are further categorized into:
//
//   - Empty key: test with k == Empty
//   - Private keys: test with k.IsPrivate() or num, ok = k.Private()
//   - Reserved keys: test with k.IsReserved(); includes the Invalid key
func (k *Key) IsSpecial() bool {
	return bytes.HasPrefix(k.object[:], specialPrefix)
}

func (k *Key) specialKind() byte {
	return k.object[len(specialPrefix)]
}

// IsPrivate returns true if the key is private. Private keys can be
// used to store 64 bits of data in them, but cannot be stored
// persistently. See Private, NewKeyPrivate, NewKeyPrivateNum.
func (k *Key) IsPrivate() bool {
	return k.IsSpecial() && k.specialKind() == 0xFF
}

// Private extracts the private data from a Key in the private range.
// The return value ok is false if the Key was not a private key.
func (k *Key) Private() (num uint64, ok bool) {
	if !k.IsPrivate() {
		return 0, false
	}

	num = binary.BigEndian.Uint64(k.object[len(specialPrefix)+1:])
	return num, true
}

// IsReserved returns true if the key is reserved. Caller should never
// use reserved keys.
func (k *Key) IsReserved() bool {
	return k.IsSpecial() && k.specialKind() != 0xFF && *k != Empty
}

func (k *Key) Unmarshal(b []byte) error {
	if len(b) != KeySize {
		return &BadKeySizeError{Key: b}
	}
	*k = NewKey(b)
	if *k == Invalid {
		return errors.New("invalid key")
	}
	return nil
}

var _ = proto.Unmarshaler(&Key{})

func (k *Key) MarshalTo(data []byte) (n int, err error) {
	n = copy(data, k.object[:])
	return n, nil
}

var _ = pb.Marshaler(&Key{})

func (*Key) Size() int {
	return KeySize
}

var _ = proto.Sizer(&Key{})

func newKey(b []byte) Key {
	k := Key{}
	n := copy(k.object[:], b)
	if n != KeySize {
		panic(BadKeySizeError{Key: b})
	}
	return k
}

// NewKey makes a new Key with the given byte contents.
// If the input happens to be a reserved byte sequence,
// the returned key will be Invalid.
//
// This function is intended for use when unmarshaling keys from
// storage.
//
// panics with BadKeySizeError if len(b) does not match KeySize
func NewKey(b []byte) Key {
	k := newKey(b)
	if bytes.HasPrefix(k.object[:], specialPrefix) &&
		k != Empty {
		return Invalid
	}
	return k
}

// NewKeyPrivate is like NewKey, but allows byte sequences in the
// private range. The private data can be extracted with the method
// Private.
//
// panics with BadKeySize  if len(b) does not match KeySize
func NewKeyPrivate(b []byte) Key {
	k := newKey(b)
	if bytes.HasPrefix(k.object[:], specialPrefix) &&
		k.object[len(specialPrefix)] != 0xFF &&
		k != Empty {
		return Invalid
	}
	return k
}

// NewKeyPrivateNum makes a new Key in the private range, with the
// given number encoded in it. The number can be extracted from the
// key with the method Private.
func NewKeyPrivateNum(num uint64) Key {
	k := Key{}
	copy(k.object[:], specialPrefix)
	k.object[len(specialPrefix)] = 0xFF
	binary.BigEndian.PutUint64(k.object[len(specialPrefix)+1:], num)
	return k
}

// The Empty key is special, it denotes no data stored (potentially
// after discarding trailing zeroes in the data). The empty key is
// all zero bytes.
var Empty = Key{}

// The Invalid key is special, it denotes input to NewKey that
// contained to use a reserved or private key. Or, respectively, just
// reserved for NewKeyPrivate.
//
// These key byte values are never marshaled anywhere, so seeing them
// in input is always illegal.
var Invalid = Key{newInvalidKey()}

func newInvalidKey() [KeySize]byte {
	var suffix = [...]byte{0xFE, 0x0B, 0xAD, 0xBA, 0xDB, 0xAD, 0xBA, 0xDB, 0xAD}
	var buf [KeySize]byte
	copy(buf[KeySize-len(suffix):], suffix[:])
	return buf
}
