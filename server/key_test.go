package server

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/agl/ed25519"
)

// Ed25519 public key is the latter half of the private key, but let's
// double-check that assumption.
func TestExtractEd25519PublicKey(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := extractEd25519Pubkey(priv), pub; !bytes.Equal(g[:], e[:]) {
		t.Errorf("assumed public key would be in private key: %x != %x", g, e)
	}
}
