package peer

import (
	"encoding"
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/agl/ed25519"
)

type ID uint32

type PublicKey [ed25519.PublicKeySize]byte

var _ encoding.BinaryMarshaler = (*PublicKey)(nil)

func (p *PublicKey) MarshalBinary() (data []byte, err error) {
	return p[:], nil
}

var _ encoding.BinaryUnmarshaler = (*PublicKey)(nil)

func (p *PublicKey) UnmarshalBinary(data []byte) error {
	if len(data) != len(p) {
		return fmt.Errorf("peer public key must be exactly %d bytes", ed25519.PublicKeySize)
	}
	copy(p[:], data)
	return nil
}

var _ flag.Value = (*PublicKey)(nil)

func (k *PublicKey) String() string {
	return hex.EncodeToString(k[:])
}

func (k *PublicKey) Set(value string) error {
	if hex.DecodedLen(len(value)) != ed25519.PublicKeySize {
		return fmt.Errorf("not a valid public key: wrong size")
	}
	if _, err := hex.Decode(k[:], []byte(value)); err != nil {
		return fmt.Errorf("not a valid public key: %v", err)
	}
	return nil
}
