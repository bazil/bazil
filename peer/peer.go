package peer

import (
	"encoding/hex"
	"flag"
	"fmt"

	"github.com/agl/ed25519"
)

type ID uint32

type Peer struct {
	ID  ID
	Pub *[ed25519.PublicKeySize]byte
}

type PublicKey [ed25519.PublicKeySize]byte

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
