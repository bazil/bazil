package control_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/tokens"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func checkHasNoPeers(app *server.App) error {
	check := func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(tokens.BucketPeer))
		c := b.Cursor()
		k, _ := c.First()
		if k != nil {
			return fmt.Errorf("did not expect stored public key: %x", k)
		}
		return nil
	}
	if err := app.DB.DB.View(check); err != nil {
		return err
	}
	return nil
}

func TestPeerAdd(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	pub := peer.PublicKey{1, 2, 3, 4, 5}
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

	getPeer := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(&pub)
		if err != nil {
			t.Fatalf("checking stored peer: %v", err)
		}
		if g, e := *p.Pub(), pub; g != e {
			t.Errorf("wrong public key stored: %x != %x", g, e)
		}
		return nil
	}
	if err := app.DB.View(getPeer); err != nil {
		t.Fatal(err)
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
	if err := checkRPCError(err, codes.InvalidArgument, "bad peer public key: peer public key must be exactly 32 bytes"); err != nil {
		t.Error(err)
	}
	if err := checkHasNoPeers(app); err != nil {
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
	if err := checkRPCError(err, codes.InvalidArgument, "bad peer public key: peer public key must be exactly 32 bytes"); err != nil {
		t.Error(err)
	}
	if err := checkHasNoPeers(app); err != nil {
		t.Error(err)
	}
}

func TestPeerAddBadPubSelf(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	addReq := &wire.PeerAddRequest{
		Pub: app.Keys.Sign.Pub[:],
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
		t.Fatalf("expected error from PeerAdd with its own public key")
	}
	if err := checkRPCError(err, codes.InvalidArgument, "cannot add self as peer"); err != nil {
		t.Error(err)
	}
	if err := checkHasNoPeers(app); err != nil {
		t.Error(err)
	}
}
