package peer_test

import (
	"testing"

	"github.com/agl/ed25519"

	"bazil.org/bazil/peer"
)

func TestPublicKeyString(t *testing.T) {
	pub := peer.PublicKey{
		0x4d, 0x0e, 0x62, 0x5e, 0xff, 0x41, 0x00, 0x6a,
		0x18, 0xb3, 0xbf, 0xda, 0x35, 0xb1, 0x40, 0xfc,
		0xad, 0x91, 0x78, 0xfe, 0x6f, 0x6c, 0xaf, 0x74,
		0xf5, 0x05, 0x9c, 0x35, 0xf0, 0xfe, 0x21, 0xaf,
	}
	if g, e := pub.String(), "4d0e625eff41006a18b3bfda35b140fcad9178fe6f6caf74f5059c35f0fe21af"; g != e {
		t.Errorf("wrong pubkey output: %q != %q", g, e)
	}
}

func TestPublicKeySet(t *testing.T) {
	var pub peer.PublicKey
	err := pub.Set("4d0e625eff41006a18b3bfda35b140fcad9178fe6f6caf74f5059c35f0fe21af")
	if err != nil {
		t.Errorf("Set: %v", err)
	}
	want := [ed25519.PublicKeySize]byte{
		0x4d, 0x0e, 0x62, 0x5e, 0xff, 0x41, 0x00, 0x6a,
		0x18, 0xb3, 0xbf, 0xda, 0x35, 0xb1, 0x40, 0xfc,
		0xad, 0x91, 0x78, 0xfe, 0x6f, 0x6c, 0xaf, 0x74,
		0xf5, 0x05, 0x9c, 0x35, 0xf0, 0xfe, 0x21, 0xaf,
	}
	if g, e := pub, want; g != e {
		t.Errorf("wrong pubkey value: %q != %q", g, e)
	}
}

func TestPublicKeySetBadShort(t *testing.T) {
	var pub peer.PublicKey
	err := pub.Set("42")
	if err == nil {
		t.Fatal("expected an error from Set")
	}
	if g, e := err.Error(), "not a valid public key: wrong size"; g != e {
		t.Errorf("wrong error message: %q != %q", g, e)
	}
}
