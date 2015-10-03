package fs_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/net/context"

	"bazil.org/bazil/db"
	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control/controltest"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/server/http/httptest"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
)

// createAndConnectVolume creates a volume on app1 and connects app2
// to it. It does all the necessary setup to authorize client to use
// resources on the server.
func createAndConnectVolume(t testing.TB, app1 *server.App, volumeName1 string, app2 *server.App, volumeName2 string) {
	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)
	pub2 := (*peer.PublicKey)(app2.Keys.Sign.Pub)

	sharingKey := [32]byte{1, 2, 3, 4, 5}

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
		if _, err := tx.Peers().Make(pub1); err != nil {
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
}

// connectVolumeOnly connects a volume.
func connectVolume(t testing.TB, app1 *server.App, volumeName1 string, app2 *server.App, volumeName2 string) {
	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)
	pub2 := (*peer.PublicKey)(app2.Keys.Sign.Pub)

	setup1 := func(tx *db.Tx) error {
		peer, err := tx.Peers().Make(pub2)
		if err != nil {
			return err
		}
		if err := peer.Storage().Allow("local"); err != nil {
			return err
		}
		v, err := tx.Volumes().GetByName(volumeName1)
		if err != nil {
			return err
		}
		if err := peer.Volumes().Allow(v); err != nil {
			return err
		}
		return nil
	}
	if err := app1.DB.Update(setup1); err != nil {
		t.Fatalf("app1 setup: %v", err)
	}

	setup2 := func(tx *db.Tx) error {
		if _, err := tx.Peers().Make(pub1); err != nil {
			return err
		}
		v, err := tx.Volumes().GetByName(volumeName2)
		if err != nil {
			return err
		}
		sharingKey, err := tx.SharingKeys().Get("friends")
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
}

// setLocation sets the location of peer identified by pub in app to loc.
func setLocation(t testing.TB, app *server.App, pub *[32]byte, loc net.Addr) {
	setLoc := func(tx *db.Tx) error {
		p, err := tx.Peers().Get((*peer.PublicKey)(pub))
		if err != nil {
			return err
		}
		if err := p.Locations().Set(loc.String()); err != nil {
			return err
		}
		return nil
	}
	if err := app.DB.Update(setLoc); err != nil {
		t.Fatalf("setup location: %v", err)
	}
}

func TestSyncSimple(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

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

func TestSyncOpen(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()

	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()

	const (
		filename = "greeting"
		input    = "hello, world"
	)
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

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

	f, err := os.Open(path.Join(mnt2.Dir, filename))
	if err != nil {
		t.Fatalf("cannot open file: %v", err)
	}
	defer f.Close()

	{
		var buf [1000]byte
		n, err := f.ReadAt(buf[:], 0)
		if err != nil && err != io.EOF {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf[:n]), input; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}

	const input2 = "goodbye, world"
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input2), 0644); err != nil {
		t.Fatalf("cannot update file: %v", err)
	}

	// sync again
	if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
		t.Fatalf("error while syncing: %v", err)
	}

	{
		// still the original content
		var buf [1000]byte
		n, err := f.ReadAt(buf[:], 0)
		if err != nil && err != io.EOF {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf[:n]), input; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}

	f.Close()

	// after the close, new content should be merged in
	//
	// TODO observing the results is racy :(
	time.Sleep(500 * time.Millisecond)

	buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}
	if g, e := string(buf), input2; g != e {
		t.Fatalf("wrong content: %q != %q", g, e)
	}

}

func TestSyncDelete(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	const (
		filename = "greeting"
		input    = "hello, world"
	)
	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

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

	if err := os.Remove(path.Join(mnt1.Dir, filename)); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// sync again
	if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
		t.Fatalf("error while syncing: %v", err)
	}

	mnt := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt.Close()
	fi, err := os.Stat(path.Join(mnt.Dir, filename))
	switch {
	case os.IsNotExist(err):
		// nothing
	case err == nil:
		t.Fatalf("file should have been removed: mode=%v size=%v", fi.Mode(), fi.Size())
	default:
		t.Fatalf("wrong error statting deleted file: %v", err)
	}
}

func TestSyncDeleteLater(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	const (
		filename = "greeting"
		input    = "hello, world"
	)
	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

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

	const input2 = "goodbye, world"
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input2), 0644); err != nil {
		t.Fatalf("cannot update file: %v", err)
	}

	// sync again
	if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
		t.Fatalf("error while syncing: %v", err)
	}

	if err := os.Remove(path.Join(mnt1.Dir, filename)); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// sync again
	if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
		t.Fatalf("error while syncing: %v", err)
	}

	mnt := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt.Close()
	if _, err := os.Stat(path.Join(mnt.Dir, filename)); !os.IsNotExist(err) {
		t.Fatalf("file should have been removed")
	}
}

func TestSyncDeleteActive(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	const (
		filename = "greeting"
		input    = "hello, world"
	)
	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()

	{
		proto, err := mnt2.Protocol()
		if err != nil {
			t.Errorf("error getting FUSE protocol version: %v", err)
		}
		if !proto.HasInvalidate() {
			t.Skip("Old FUSE protocol")
		}
	}

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

	buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}
	if g, e := string(buf), input; g != e {
		t.Fatalf("wrong content: %q != %q", g, e)
	}

	if err := os.Remove(path.Join(mnt1.Dir, filename)); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// sync again
	if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
		t.Fatalf("error while syncing: %v", err)
	}

	if _, err := os.Stat(path.Join(mnt2.Dir, filename)); !os.IsNotExist(err) {
		t.Fatalf("file should have been removed")
	}
}

func TestSyncRoundtrip(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)
	pub2 := (*peer.PublicKey)(app2.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)
	connectVolume(t, app2, volumeName2, app1, volumeName1)

	var wg sync.WaitGroup
	defer wg.Wait()

	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	web2 := httptest.ServeHTTP(t, &wg, app2)
	defer web2.Close()
	setLocation(t, app1, app2.Keys.Sign.Pub, web2.Addr())

	const (
		filename = "greeting"
		input1   = "hello, world"
		input2   = "goodbye"
	)
	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()

	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input1), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// trigger sync
	ctrl2 := controltest.ListenAndServe(t, &wg, app2)
	defer ctrl2.Close()
	rpcConn2, err := grpcunix.Dial(filepath.Join(app2.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn2.Close()
	rpcClient2 := wire.NewControlClient(rpcConn2)
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName2,
			Pub:        pub1[:],
		}
		if _, err := rpcClient2.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read file after sync: %v", err)
	}
	if g, e := string(buf), input1; g != e {
		t.Errorf("wrong contents after sync: %q != %q", g, e)
	}
	if err := ioutil.WriteFile(path.Join(mnt2.Dir, filename), []byte(input2), 0644); err != nil {
		t.Fatalf("cannot update file: %v", err)
	}

	// trigger sync the other way
	ctrl1 := controltest.ListenAndServe(t, &wg, app1)
	defer ctrl1.Close()
	rpcConn1, err := grpcunix.Dial(filepath.Join(app1.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn1.Close()
	rpcClient1 := wire.NewControlClient(rpcConn1)
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName1,
			Pub:        pub2[:],
		}
		if _, err := rpcClient1.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	buf, err = ioutil.ReadFile(path.Join(mnt1.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read pending entry: %v", err)
	}
	if g, e := string(buf), input2; g != e {
		t.Errorf("wrong contents after second sync: %q != %q", g, e)
	}
}

func TestSyncRename(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)

	var wg sync.WaitGroup
	defer wg.Wait()
	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()

	const (
		filename1 = "greeting"
		filename2 = "cheers"
		input     = "hello, world"
	)

	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename1), []byte(input), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// trigger sync
	ctrl := controltest.ListenAndServe(t, &wg, app2)
	defer ctrl.Close()
	rpcConn, err := grpcunix.Dial(filepath.Join(app2.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn.Close()
	rpcClient := wire.NewControlClient(rpcConn)
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName2,
			Pub:        pub1[:],
		}
		if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	{
		buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename1))
		if err != nil {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf), input; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}

	{
		check := map[string]bazfstestutil.FileInfoCheck{
			filename1: nil,
		}
		if err := bazfstestutil.CheckDir(mnt2.Dir, check); err != nil {
			t.Error(err)
		}
	}

	// rename the original
	if err := os.Rename(path.Join(mnt1.Dir, filename1), path.Join(mnt1.Dir, filename2)); err != nil {
		t.Fatal(err)
	}

	// sync again
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName2,
			Pub:        pub1[:],
		}
		if _, err := rpcClient.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	{
		check := map[string]bazfstestutil.FileInfoCheck{
			filename2: nil,
		}
		if err := bazfstestutil.CheckDir(mnt2.Dir, check); err != nil {
			t.Error(err)
		}
	}

	{
		buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename2))
		if err != nil {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf), input; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}
}

func TestSyncRenameWithResolvedConflict(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app1 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app1"), "1")
	defer app1.Close()
	app2 := bazfstestutil.NewAppWithName(t, tmp.Subdir("app2"), "2")
	defer app2.Close()

	pub1 := (*peer.PublicKey)(app1.Keys.Sign.Pub)
	pub2 := (*peer.PublicKey)(app2.Keys.Sign.Pub)

	const (
		volumeName1 = "testvol1"
		volumeName2 = "testvol2"
	)
	createAndConnectVolume(t, app1, volumeName1, app2, volumeName2)
	connectVolume(t, app2, volumeName2, app1, volumeName1)

	var wg sync.WaitGroup
	defer wg.Wait()

	web1 := httptest.ServeHTTP(t, &wg, app1)
	defer web1.Close()
	setLocation(t, app2, app1.Keys.Sign.Pub, web1.Addr())

	web2 := httptest.ServeHTTP(t, &wg, app2)
	defer web2.Close()
	setLocation(t, app1, app2.Keys.Sign.Pub, web2.Addr())

	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()

	const (
		filename1 = "greeting"
		filename2 = "cheers"
		input1    = "hello, world"
		input2    = "goodbye"
		input3    = "farewell"
	)

	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename1), []byte(input1), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// trigger sync
	ctrl2 := controltest.ListenAndServe(t, &wg, app2)
	defer ctrl2.Close()
	rpcConn2, err := grpcunix.Dial(filepath.Join(app2.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn2.Close()
	rpcClient2 := wire.NewControlClient(rpcConn2)
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName2,
			Pub:        pub1[:],
		}
		if _, err := rpcClient2.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	{
		buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename1))
		if err != nil {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf), input1; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}

	{
		check := map[string]bazfstestutil.FileInfoCheck{
			filename1: nil,
		}
		if err := bazfstestutil.CheckDir(mnt2.Dir, check); err != nil {
			t.Error(err)
		}
	}

	// cause a conflict
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename1), []byte(input2), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}
	if err := ioutil.WriteFile(path.Join(mnt2.Dir, filename1), []byte(input3), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	// sync backward
	ctrl1 := controltest.ListenAndServe(t, &wg, app1)
	defer ctrl1.Close()
	rpcConn1, err := grpcunix.Dial(filepath.Join(app1.DataDir, "control"))
	if err != nil {
		t.Fatal(err)
	}
	defer rpcConn1.Close()
	rpcClient1 := wire.NewControlClient(rpcConn1)
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName1,
			Pub:        pub2[:],
		}
		if _, err := rpcClient1.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	{
		buf, err := ioutil.ReadFile(path.Join(mnt1.Dir, filename1))
		if err != nil {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf), input2; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}

	var seen os.FileInfo
	entryCheckers := map[string]bazfstestutil.FileInfoCheck{
		"": func(fi os.FileInfo) error {
			if seen != nil {
				return fmt.Errorf("expected only one file, already saw %q", seen.Name())
			}
			seen = fi
			return nil
		},
	}
	if err := bazfstestutil.CheckDir(path.Join(mnt1.Dir, ".bazil", "pending", filename1), entryCheckers); err != nil {
		t.Error(err)
	}
	if seen == nil {
		t.Fatal("expected to see a pending clock")
	}

	// declare the conflict handled
	if err := os.Remove(path.Join(mnt1.Dir, ".bazil", "pending", filename1, seen.Name())); err != nil {
		t.Fatalf("error removing pending entry: %v", err)
	}

	// rename the original
	if err := os.Rename(path.Join(mnt1.Dir, filename1), path.Join(mnt1.Dir, filename2)); err != nil {
		t.Fatal(err)
	}

	// sync again
	{
		ctx := context.Background()
		req := &wire.VolumeSyncRequest{
			VolumeName: volumeName2,
			Pub:        pub1[:],
		}
		if _, err := rpcClient2.VolumeSync(ctx, req); err != nil {
			t.Fatalf("error while syncing: %v", err)
		}
	}

	{
		check := map[string]bazfstestutil.FileInfoCheck{
			filename2: nil,
		}
		if err := bazfstestutil.CheckDir(mnt2.Dir, check); err != nil {
			t.Error(err)
		}
	}

	{
		buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename2))
		if err != nil {
			t.Fatalf("cannot read file: %v", err)
		}
		if g, e := string(buf), input2; g != e {
			t.Fatalf("wrong content: %q != %q", g, e)
		}
	}
}
