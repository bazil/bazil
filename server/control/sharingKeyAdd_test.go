package control

import (
	"fmt"
	"path/filepath"
	"testing"

	"bazil.org/bazil/db"
	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
	"golang.org/x/net/context"
	"google.golang.org/grpc/codes"
)

func checkNoSharingKey(name string) func(tx *db.Tx) error {
	check := func(tx *db.Tx) error {
		key, err := tx.SharingKeys().Get("foo")
		switch err {
		case db.ErrSharingKeyNotFound:
			// nothing
		case nil:
			return fmt.Errorf("secret stored even on error: %x", key)
		default:
			return fmt.Errorf("error checking sharing key: %v", err)
		}
		return nil
	}
	return check
}

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
	check := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Get("foo")
		if err != nil {
			t.Fatalf("error checking sharing key: %v", err)
		}
		var key [32]byte
		sharingKey.Secret(&key)
		if g, e := key, secret; g != e {
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
	if err := checkRPCError(err, codes.InvalidArgument, "invalid sharing key name"); err != nil {
		t.Error(err)
	}

	if err := app.DB.View(checkNoSharingKey("foo")); err != nil {
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
	if err := checkRPCError(err, codes.InvalidArgument, "sharing key must be exactly 32 bytes"); err != nil {
		t.Error(err)
	}

	if err := app.DB.View(checkNoSharingKey("foo")); err != nil {
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
	if err := checkRPCError(err, codes.InvalidArgument, "sharing key must be exactly 32 bytes"); err != nil {
		t.Error(err)
	}

	if err := app.DB.View(checkNoSharingKey("foo")); err != nil {
		t.Error(err)
	}
}
