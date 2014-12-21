package peer

import (
	"github.com/agl/ed25519"
)

type ID uint32

type Peer struct {
	ID  ID
	Pub *[ed25519.PublicKeySize]byte
}
