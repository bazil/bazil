package fs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"

	"golang.org/x/net/context"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/controltest"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/server/http/httptest"
	"bazil.org/bazil/util/grpcunix"
	"bazil.org/bazil/util/tempdir"
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

	if err := bazfstestutil.CheckDir(p, nil); err != nil {
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

	listCheckers := map[string]bazfstestutil.FileInfoCheck{
		filename: func(fi os.FileInfo) error {
			if g, e := fi.Mode(), os.ModeDir|0500; g != e {
				return fmt.Errorf("wrong mode: %v != %v", g, e)
			}
			return nil
		},
	}
	if err := bazfstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending"), listCheckers); err != nil {
		t.Error(err)
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
	if err := bazfstestutil.CheckDir(path.Join(mnt2.Dir, ".bazil", "pending", filename), entryCheckers); err != nil {
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
