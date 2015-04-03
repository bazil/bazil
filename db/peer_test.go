package db_test

import (
	"fmt"
	"testing"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
)

func checkMakePeer(tx *db.Tx, pub *peer.PublicKey, id peer.ID) error {
	peer, err := tx.Peers().Make(pub)
	if err != nil {
		return fmt.Errorf("unexpected peers.Make error: %v", err)
	}
	if g, e := *peer.Pub(), *pub; g != e {
		return fmt.Errorf("peer pubkey came back wrong: %v != %v", g, e)
	}
	if g, e := peer.ID(), id; g != e {
		return fmt.Errorf("wrong peer ID: %v != %v", g, e)
	}
	return nil
}

func TestGetPeerNotFound(t *testing.T) {
	DB := NewTestDB(t)
	defer DB.Close()

	pub1 := &peer.PublicKey{0x42, 0x42, 0x42}
	get := func(tx *db.Tx) error {
		peer, err := tx.Peers().Get(pub1)
		if g, e := err, db.ErrPeerNotFound; g != e {
			t.Errorf("expected ErrPeerNotFound, got %v", err)
		}
		if peer != nil {
			t.Errorf("peer should be nil on error: %v", peer)
		}
		return nil
	}
	if err := DB.View(get); err != nil {
		t.Fatal(err)
	}
}

func TestMakePeer(t *testing.T) {
	DB := NewTestDB(t)
	defer DB.Close()

	pub1 := &peer.PublicKey{0x42, 0x42, 0x42}
	pub2 := &peer.PublicKey{0xC0, 0xFF, 0xEE}

	check := func(tx *db.Tx) error {
		if err := checkMakePeer(tx, pub1, 1); err != nil {
			t.Error(err)
		}
		if err := checkMakePeer(tx, pub1, 1); err != nil {
			t.Error(err)
		}
		if err := checkMakePeer(tx, pub2, 2); err != nil {
			t.Error(err)
		}
		if err := checkMakePeer(tx, pub1, 1); err != nil {
			t.Error(err)
		}
		return nil
	}
	if err := DB.Update(check); err != nil {
		t.Fatal(err)
	}
}
