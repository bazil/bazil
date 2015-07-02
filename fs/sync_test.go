package fs_test

import (
	"io/ioutil"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/net/context"

	"bazil.org/bazil/db"
	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/controltest"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/server/http/httptest"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
)

func TestSyncSimple(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewApp(t, tmp.Subdir("app1"))
	defer app1.Close()
	app2 := bazfstestutil.NewApp(t, tmp.Subdir("app2"))
	defer app2.Close()

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)
	pub2 := (*peer.PublicKey)(app2.Keys.Sign.Pub)

	sharingKey := [32]byte{1, 2, 3, 4, 5}

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	var volID db.VolumeID
	setup1 := func(tx *db.Tx) error {
		peer, err := tx.Peers().Make(pub2)
		if err != nil {
			return err
		}
		if err := peer.Storage().Allow("local"); err != nil {
			return err
		}
		sharingKey, err := tx.SharingKeys().Add("friends", &sharingKey)
		if err != nil {
			return err
		}
		v, err := tx.Volumes().Create(volumeName1, "local", sharingKey)
		if err != nil {
			return err
		}
		if err := peer.Volumes().Allow(v); err != nil {
			return err
		}
		v.VolumeID(&volID)
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
		sharingKey, err := tx.SharingKeys().Add("friends", &sharingKey)
		if err != nil {
			return err
		}
		v, err := tx.Volumes().Add(volumeName2, &volID, "local", sharingKey)
		if err != nil {
			return err
		}
		if err := v.Storage().Add("jdoe", "peerkey:"+pub1.String(), sharingKey); err != nil {
			return err
		}
		return nil
	}
	if err := app2.DB.Update(setup2); err != nil {
		t.Fatalf("app2 setup location: %v", err)
	}

	const (
		filename = "greeting"
		input    = "hello, world"
	)
	func() {
		mnt := bazfstestutil.Mounted(t, app1, volumeName1)
		defer mnt.Close()
		if err := ioutil.WriteFile(path.Join(mnt.Dir, filename), []byte(input), 0644); err != nil {
			t.Fatalf("cannot create file: %v", err)
		}
	}()

	// trigger sync
	ctrl := controltest.ListenAndServe(t, &wg, app2)
	defer ctrl.Close()
	rpcConn, err := grpcunix.Dial(filepath.Join(app2.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)
	ctx := context.Background()
	req := &wire.VolumeSyncRequest{
		VolumeName: volumeName2,
		Pub:        pub1[:],
	}
	if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
		t.Fatalf("error while syncing: %v", err)
	}

	mnt := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt.Close()
	buf, err := ioutil.ReadFile(path.Join(mnt.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}
	if g, e := string(buf), input; g != e {
		t.Fatalf("wrong content: %q != %q", g, e)
	}
}
