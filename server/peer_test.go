package server_test

import (
	"fmt"
	"testing"

	"github.com/agl/ed25519"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server"
	"bazil.org/bazil/util/tempdir"
)

func checkMakePeer(app *server.App, pub *[ed25519.PublicKeySize]byte, id peer.ID) error {
	peer, err := app.MakePeer(pub)
	if err != nil {
		return err
	}
	if g, e := *peer.Pub, *pub; g != e {
		return fmt.Errorf("peer pubkey came back wrong: %v != %v", g, e)
	}
	if g, e := peer.ID, id; g != e {
		return fmt.Errorf("wrong peer ID: %v != %v", g, e)
	}
	return nil
}

func TestGetPeerNotFound(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	pub1 := &[ed25519.PublicKeySize]byte{0x42, 0x42, 0x42}
	peer, err := app.GetPeer(pub1)
	if g, e := err, server.ErrPeerNotFound; g != e {
		t.Errorf("expected ErrPeerNotFound, got %v", err)
	}
	if peer != nil {
		t.Errorf("peer should be nil on error: %v", peer)
	}
}

func TestMakePeer(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()

	pub1 := &[ed25519.PublicKeySize]byte{0x42, 0x42, 0x42}
	pub2 := &[ed25519.PublicKeySize]byte{0xC0, 0xFF, 0xEE}

	if err := checkMakePeer(app, pub1, 1); err != nil {
		t.Error(err)
	}
	if err := checkMakePeer(app, pub1, 1); err != nil {
		t.Error(err)
	}
	if err := checkMakePeer(app, pub2, 2); err != nil {
		t.Error(err)
	}
	if err := checkMakePeer(app, pub1, 1); err != nil {
		t.Error(err)
	}
}
