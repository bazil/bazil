package control

import (
	"bytes"
	"path/filepath"
	"testing"

	"bazil.org/bazil/control/wire"
	"bazil.org/bazil/server"
	"bazil.org/bazil/tokens"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func TestSharingAdd(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	secret := [32]byte{1, 2, 3, 4, 5}
	addReq := &wire.SharingKeyAddRequest{
		Name:   "foo",
		Secret: secret[:],
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	if _, err := rpcClient.SharingKeyAdd(ctx, addReq); err != nil {
		t.Fatalf("adding sharing key failed: %v", err)
	}
	check := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketSharing))
		val := bucket.Get([]byte("foo"))
		if g, e := val, secret; !bytes.Equal(g[:], e[:]) {
			t.Errorf("wrong secret stored: %x != %x", g, e)
		}
		return nil
	}
	if err := app.DB.View(check); err != nil {
		t.Error(err)
	}
}

func TestSharingAddBadNameEmpty(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	secret := [32]byte{1, 2, 3, 4, 5}
	addReq := &wire.SharingKeyAddRequest{
		Name:   "",
		Secret: secret[:],
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	_, err = rpcClient.SharingKeyAdd(ctx, addReq)
	if err == nil {
		t.Fatalf("expected error from SharingKeyAdd with empty name")
	}
	checkRPCError(t, err, codes.InvalidArgument, "invalid sharing key name")

	check := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketSharing))
		val := bucket.Get([]byte("foo"))
		if g := val; g != nil {
			t.Errorf("secret stored even on error: %x", g)
		}
		return nil
	}
	if err := app.DB.View(check); err != nil {
		t.Error(err)
	}
}

func TestSharingAddBadSecretLong(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	tooLong := make([]byte, 33)
	addReq := &wire.SharingKeyAddRequest{
		Name:   "foo",
		Secret: tooLong,
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	_, err = rpcClient.SharingKeyAdd(ctx, addReq)
	if err == nil {
		t.Fatalf("expected error from SharingKeyAdd with too long secret")
	}
	checkRPCError(t, err, codes.InvalidArgument, "sharing key must be exactly 32 bytes")

	check := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketSharing))
		val := bucket.Get([]byte("foo"))
		if g := val; g != nil {
			t.Errorf("secret stored even on error: %x", g)
		}
		return nil
	}
	if err := app.DB.View(check); err != nil {
		t.Error(err)
	}
}

func TestSharingAddBadSecretShort(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app, err := server.New(tmp.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer controlListenAndServe(t, app)()

	tooShort := make([]byte, 33)
	addReq := &wire.SharingKeyAddRequest{
		Name:   "foo",
		Secret: tooShort,
	}

	rpcConn, err := grpcunix.Dial(filepath.Join(app.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)

	ctx := context.Background()
	_, err = rpcClient.SharingKeyAdd(ctx, addReq)
	if err == nil {
		t.Fatalf("expected error from SharingKeyAdd with too short secret")
	}
	checkRPCError(t, err, codes.InvalidArgument, "sharing key must be exactly 32 bytes")

	check := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketSharing))
		val := bucket.Get([]byte("foo"))
		if g := val; g != nil {
			t.Errorf("secret stored even on error: %x", g)
		}
		return nil
	}
	if err := app.DB.View(check); err != nil {
		t.Error(err)
	}
}
