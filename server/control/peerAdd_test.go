package control

import (
	"path/filepath"
	"testing"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
	"github.com/agl/ed25519"
	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func TestPeerAdd(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	pub := [ed25519.PublicKeySize]byte{1, 2, 3, 4, 5}
	addReq := &wire.PeerAddRequest{
		Pub: pub[:],
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	if _, err := rpcClient.PeerAdd(ctx, addReq); err != nil {
		t.Fatalf("adding peer failed: %v", err)
	}

	p, err := app.GetPeer(&pub)
	if err != nil {
		t.Fatalf("checking stored peer: %v", err)
	}
	if g, e := *p.Pub, pub; g != e {
		t.Errorf("wrong public key stored: %x != %x", g, e)
	}
}

func TestPeerAddBadPubLong(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	tooLong := make([]byte, 33)
	addReq := &wire.PeerAddRequest{
		Pub: tooLong,
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	_, err = rpcClient.PeerAdd(ctx, addReq)
	if err == nil {
		t.Fatalf("expected error from PeerAdd with too long public key")
	}
	checkRPCError(t, err, codes.InvalidArgument, "peer public key must be exactly 32 bytes")

	check := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketPeer))
		val := bucket.Get([]byte("foo"))
		if g := val; g != nil {
			t.Errorf("public key stored even on error: %x", g)
		}
		return nil
	}
	if err := app.DB.View(check); err != nil {
		t.Error(err)
	}
}

func TestPeerAddBadPubShort(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	tooShort := make([]byte, 33)
	addReq := &wire.PeerAddRequest{
		Pub: tooShort,
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	_, err = rpcClient.PeerAdd(ctx, addReq)
	if err == nil {
		t.Fatalf("expected error from PeerAdd with too short public key")
	}
	checkRPCError(t, err, codes.InvalidArgument, "peer public key must be exactly 32 bytes")

	check := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketPeer))
		val := bucket.Get([]byte("foo"))
		if g := val; g != nil {
			t.Errorf("public key stored even on error: %x", g)
		}
		return nil
	}
	if err := app.DB.View(check); err != nil {
		t.Error(err)
	}
}
