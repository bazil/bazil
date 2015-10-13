package peer_test

import (
	"sync"
	"testing"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"bazil.org/bazil/db"
	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server/http/httptest"
	"bazil.org/bazil/util/tempdir"
)

func TestPing(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)
	pub2 := (*peer.PublicKey)(app2.Keys.Sign.Pub)

	setup1 := func(tx *db.Tx) error {
		if _, err := tx.Peers().Make(pub2); err != nil {
			return err
		}
		return nil
	}
	if err := app1.DB.Update(setup1); err != nil {
		t.Fatalf("app1 setup: %v", err)
	}

	setup2 := func(tx *db.Tx) error {
		p, err := tx.Peers().Make(pub1)
		if err != nil {
			return err
		}
		if err := p.Locations().Set(web1.Addr().String()); err != nil {
			return err
		}
		return nil
	}
	if err := app2.DB.Update(setup2); err != nil {
		t.Fatalf("app2 setup location: %v", err)
	}

	client, err := app2.DialPeer(pub1)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	if _, err := client.Ping(ctx, &wire.PingRequest{}); err != nil {
		t.Errorf("ping failed: %v", err)
	}
}

func TestPingBadNotPeer(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	setup2 := func(tx *db.Tx) error {
		p, err := tx.Peers().Make(pub1)
		if err != nil {
			return err
		}
		if err := p.Locations().Set(web1.Addr().String()); err != nil {
			return err
		}
		return nil
	}
	if err := app2.DB.Update(setup2); err != nil {
		t.Fatalf("app2 setup: %v", err)
	}

	client, err := app2.DialPeer(pub1)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	if _, err := client.Ping(ctx, &wire.PingRequest{}); grpc.Code(err) != codes.PermissionDenied {
		t.Errorf("wrong error from ping: %v", err)
	}
}
