package fs_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/controltest"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/server/http/httptest"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
	"bazil.org/fuse/fs/fstestutil"
)

func TestPendingListEmpty(t *testing.T) {
	tmp := tempdir.New(t)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(t, tmp.Subdir("data"))
	defer app.Close()
	bazfstestutil.CreateVolume(t, app, "default")

	mnt := bazfstestutil.Mounted(t, app, "default")
	defer mnt.Close()

	p := path.Join(mnt.Dir, ".bazil", "pending")
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat of pending dir failed: %v", err)
	}
	if g, e := fi.Mode(), os.ModeDir|0500; g != e {
		t.Errorf("wrong mode: %v != %v", g, e)
	}

	if err := fstestutil.CheckDir(p, nil); err != nil {
		t.Error(err)
	}
}

func TestPendingConflict(t *testing.T) {
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
		input1   = "hello, world"
		input2   = "goodbye"
	)
	mnt1 := bazfstestutil.Mounted(t, app1, volumeName1)
	defer mnt1.Close()
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input1), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()
	if err := ioutil.WriteFile(path.Join(mnt2.Dir, filename), []byte(input2), 0644); err != nil {
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

	// the file was not changed, due to the conflict
	buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read file after sync: %v", err)
	}
	if g, e := string(buf), input2; g != e {
		t.Errorf("wrong contents after sync: %q != %q", g, e)
	}

	listCheckers := map[string]fstestutil.FileInfoCheck{
		filename: func(fi os.FileInfo) error {
			if g, e := fi.Mode(), os.ModeDir|0500; g != e {
				return fmt.Errorf("wrong mode: %v != %v", g, e)
			}
			return nil
		},
	}
	if err := fstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending"), listCheckers); err != nil {
		t.Error(err)
	}

	var seen os.FileInfo
	entryCheckers := map[string]fstestutil.FileInfoCheck{
		"": func(fi os.FileInfo) error {
			if seen != nil {
				return fmt.Errorf("expected only one file, already saw %q", seen.Name())
			}
			seen = fi
			return nil
		},
	}
	if err := fstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending", filename), entryCheckers); err != nil {
		t.Error(err)
	}
	if seen == nil {
		t.Fatal("expected to see a pending clock")
	}
	// TODO right mode
	if g, e := seen.Mode(), os.FileMode(0444); g != e {
		t.Errorf("wrong mode: %v != %v", g, e)
	}
	buf, err = ioutil.ReadFile(path.Join(mnt2.Dir, ".bazil", "pending", filename, seen.Name()))
	if err != nil {
		t.Fatalf("cannot read pending entry: %v", err)
	}
	if g, e := string(buf), input1; g != e {
		t.Errorf("wrong pending contents: %q != %q", g, e)
	}
}

func TestPendingResolveByRemove(t *testing.T) {
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
	if err := ioutil.WriteFile(path.Join(mnt1.Dir, filename), []byte(input1), 0644); err != nil {
		t.Fatalf("cannot create file: %v", err)
	}

	mnt2 := bazfstestutil.Mounted(t, app2, volumeName2)
	defer mnt2.Close()
	if err := ioutil.WriteFile(path.Join(mnt2.Dir, filename), []byte(input2), 0644); err != nil {
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

	// the file was not changed, due to the conflict
	buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename))
	if err != nil {
		t.Fatalf("cannot read file after sync: %v", err)
	}
	if g, e := string(buf), input2; g != e {
		t.Errorf("wrong contents after sync: %q != %q", g, e)
	}

	listCheckers := map[string]fstestutil.FileInfoCheck{
		filename: func(fi os.FileInfo) error {
			if g, e := fi.Mode(), os.ModeDir|0500; g != e {
				return fmt.Errorf("wrong mode: %v != %v", g, e)
			}
			return nil
		},
	}
	if err := fstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending"), listCheckers); err != nil {
		t.Error(err)
	}

	var seen os.FileInfo
	entryCheckers := map[string]fstestutil.FileInfoCheck{
		"": func(fi os.FileInfo) error {
			if seen != nil {
				return fmt.Errorf("expected only one file, already saw %q", seen.Name())
			}
			seen = fi
			return nil
		},
	}
	if err := fstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending", filename), entryCheckers); err != nil {
		t.Error(err)
	}
	if seen == nil {
		t.Fatal("expected to see a pending clock")
	}
	if err := os.Remove(path.Join(mnt2.Dir, ".bazil", "pending", filename, seen.Name())); err != nil {
		t.Fatalf("error removing pending entry: %v", err)
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

func TestPendingConflictTombstone(t *testing.T) {
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

	// cause conflict
	if err := os.Remove(path.Join(mnt1.Dir, filename)); err != nil {
		t.Fatalf("cannot remove file: %v", err)
	}
	if err := ioutil.WriteFile(path.Join(mnt2.Dir, filename), []byte(input2), 0644); err != nil {
		t.Fatalf("cannot update file: %v", err)
	}

	// trigger sync again
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

	// the file was not changed, due to the conflict
	{
		buf, err := ioutil.ReadFile(path.Join(mnt2.Dir, filename))
		if err != nil {
			t.Fatalf("cannot read file after sync: %v", err)
		}
		if g, e := string(buf), input2; g != e {
			t.Errorf("wrong contents after sync: %q != %q", g, e)
		}
	}

	listCheckers := map[string]fstestutil.FileInfoCheck{
		filename: func(fi os.FileInfo) error {
			if g, e := fi.Mode(), os.ModeDir|0500; g != e {
				return fmt.Errorf("wrong mode: %v != %v", g, e)
			}
			return nil
		},
	}
	if err := fstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending"), listCheckers); err != nil {
		t.Error(err)
	}

	var seen os.FileInfo
	entryCheckers := map[string]fstestutil.FileInfoCheck{
		"": func(fi os.FileInfo) error {
			if seen != nil {
				return fmt.Errorf("expected only one file, already saw %q", seen.Name())
			}
			seen = fi
			return nil
		},
	}
	if err := fstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending", filename), entryCheckers); err != nil {
		t.Error(err)
	}
	if seen == nil {
		t.Fatal("expected to see a pending clock")
	}
	if g, e := seen.Mode(), os.ModeSymlink|0500; g != e {
		t.Errorf("wrong mode: %v != %v", g, e)
	}
	target, err := os.Readlink(path.Join(mnt2.Dir, ".bazil", "pending", filename, seen.Name()))
	if err != nil {
		t.Fatalf("cannot read pending tombstone symlink: %v", err)
	}
	if g, e := string(target), ".deleted"; g != e {
		t.Errorf("wrong pending tombstone symlink target: %q != %q", g, e)
	}
}
