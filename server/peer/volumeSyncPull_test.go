package peer_test

import (
	"io"
	"io/ioutil"
	"path"
	"reflect"
	"sync"
	"testing"

	"golang.org/x/net/context"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/db"
	"bazil.org/bazil/fs/clock"
	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server/http/httptest"
	"bazil.org/bazil/util/tempdir"
)

func TestSyncPull(t *testing.T) {
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

	var volID db.VolumeID
	sharingKey := [32]byte{42, 42, 42, 13}
	const volumeName = "foo"

	setup1 := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Add("testkey", &sharingKey)
		if err != nil {
			return err
		}
		v, err := tx.Volumes().Create(volumeName, "local", sharingKey)
		if err != nil {
			return err
		}
		v.VolumeID(&volID)
		p, err := tx.Peers().Make(pub2)
		if err != nil {
			return err
		}
		if err := p.Volumes().Allow(v); err != nil {
			return err
		}
		if err := p.Storage().Allow("local"); err != nil {
			return err
		}
		return nil
	}
	if err := app1.DB.Update(setup1); err != nil {
		t.Fatalf("app1 setup: %v", err)
	}

	setup2 := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Add("testkey", &sharingKey)
		if err != nil {
			return err
		}
		v, err := tx.Volumes().Add(volumeName, &volID, "local", sharingKey)
		if err != nil {
			return err
		}
		v.VolumeID(&volID)
		p, err := tx.Peers().Make(pub1)
		if err != nil {
			return err
		}
		if err := v.Storage().Add("jdoe", "peerkey:"+pub1.String(), sharingKey); err != nil {
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

	var chunkStore2 chunks.Store
	openKV := func(tx *db.Tx) error {
		// This cannot be combined into setup2 because OpenKV/DialPeer
		// starts its own transaction, and wouldn't see the
		// uncommitted peer.
		v, err := tx.Volumes().GetByVolumeID(&volID)
		if err != nil {
			return err
		}
		kvstore, err := app2.OpenKV(tx, v.Storage())
		if err != nil {
			return err
		}
		chunkStore2 = kvchunks.New(kvstore)
		return nil
	}
	if err := app2.DB.View(openKV); err != nil {
		t.Fatalf("cannot open storage for app2: %v", err)
	}

	const testFileName = "greeting"
	const testFileContent = "hello, world"
	func() {
		mnt := bazfstestutil.Mounted(t, app1, volumeName)
		defer mnt.Close()
		if err := ioutil.WriteFile(path.Join(mnt.Dir, testFileName), []byte(testFileContent), 0644); err != nil {
			t.Fatalf("cannot create file: %v", err)
		}
	}()

	client, err := app2.DialPeer(pub1)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	volIDBuf, err := volID.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal volume id: %v", err)
	}
	ctx := context.Background()
	stream, err := client.VolumeSyncPull(ctx, &wire.VolumeSyncPullRequest{
		VolumeID: volIDBuf,
	})
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	item, err := stream.Recv()
	if err != nil {
		t.Fatalf("sync stream failed: %v", err)
	}
	if g, e := item.Error, wire.VolumeSyncPullItem_SUCCESS; g != e {
		t.Errorf("unexpected error: %v != %v", g, e)
	}
	if g, e := item.Peers, map[uint32][]byte{
		0: pub1[:],
		1: pub2[:],
	}; !reflect.DeepEqual(g, e) {
		t.Errorf("bad peers: %v != %v", g, e)
	}

	wantFiles := map[string]func(*wire.Dirent){
		testFileName: func(de *wire.Dirent) {
			if de.File == nil {
				t.Errorf("wrong type for %q, not a file: %v", de.Name, de)
				return
			}

			var c clock.Clock
			if err := c.UnmarshalBinary(de.Clock); err != nil {
				t.Errorf("invalid clock for %q: %v", de.Name, err)
				return
			}
			if g, e := c.String(), `{sync{0:1} mod{0:1} create{0:1}}`; g != e {
				t.Errorf("wrong clock for %q: %v != %v", de.Name, g, e)
				return
			}

			// verify file contents
			manifest, err := de.File.Manifest.ToBlob("file")
			if err != nil {
				t.Errorf("cannot open manifest for %q: %v", de.Name, err)
				return
			}
			blob, err := blobs.Open(chunkStore2, manifest)
			if err != nil {
				t.Errorf("cannot open blob for %q: %v", de.Name, err)
				return
			}
			r := io.NewSectionReader(blob, 0, int64(blob.Size()))
			buf, err := ioutil.ReadAll(r)
			if err != nil {
				t.Errorf("cannot read blob for %q: %v", de.Name, err)
				return
			}
			if g, e := string(buf), testFileContent; g != e {
				t.Errorf("wrong content for %q: %q != %q", de.Name, g, e)
			}
		},
	}
	for _, de := range item.Children {
		fn, ok := wantFiles[de.Name]
		if !ok {
			t.Errorf("unexpected direntry: %q", de.Name)
			continue
		}
		fn(de)
		delete(wantFiles, de.Name)
	}
	for name, _ := range wantFiles {
		t.Errorf("missing direntry: %q", name)
	}

	item, err = stream.Recv()
	if err != io.EOF {
		t.Errorf("expected eof, got error %v, item=%v", err, item)
	}
}

// TODO TestSyncPullBadNotPeer
// TODO TestSyncPullBadPeerNotAllowed
