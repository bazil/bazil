package untrusted_test

import (
	"context"
	"testing"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/kvchunks"
	"bazil.org/bazil/kv/kvmock"
	"bazil.org/bazil/kv/untrusted"
)

const GREETING = "Hello, world"

func TestSimple(t *testing.T) {
	remote := &kvmock.InMemory{}
	secret := &[32]byte{
		42, 42, 42, 42, 42, 42, 42, 42,
		42, 42, 42, 42, 42, 42, 42, 42,
		42, 42, 42, 42, 42, 42, 42, 42,
		42, 42, 42, 42, 42, 42, 42, 42,
	}
	converg := untrusted.New(remote, secret)
	store := kvchunks.New(converg)

	orig := &chunks.Chunk{
		Type:  "testchunk",
		Level: 3,
		Buf:   []byte(GREETING),
	}
	ctx := context.Background()
	key, err := store.Add(ctx, orig)
	if err != nil {
		t.Fatalf("store.Add failed: %v", err)
	}

	got, err := store.Get(ctx, key, "testchunk", 3)
	if err != nil {
		t.Fatalf("store.Get failed: %v", err)
	}
	if got == nil {
		t.Fatalf("store.Get gave nil chunk")
	}
	if g, e := got.Type, "testchunk"; g != e {
		t.Errorf("unexpected chunk data: %v != %v", g, e)
	}
	if g, e := got.Level, uint8(3); g != e {
		t.Errorf("unexpected chunk data: %v != %v", g, e)
	}
	if g, e := string(got.Buf), GREETING; g != e {
		t.Errorf("unexpected chunk data: %v != %v", g, e)
	}
}

func TestWrongType(t *testing.T) {
	remote := &kvmock.InMemory{}
	secret := &[32]byte{
		42, 42, 42, 42, 42, 42, 42, 42,
		42, 42, 42, 42, 42, 42, 42, 42,
		42, 42, 42, 42, 42, 42, 42, 42,
		42, 42, 42, 42, 42, 42, 42, 42,
	}
	converg := untrusted.New(remote, secret)
	store := kvchunks.New(converg)

	phony := &chunks.Chunk{
		Type:  "evilchunk",
		Level: 3,
		Buf:   []byte(GREETING),
	}
	ctx := context.Background()
	_, err := store.Add(ctx, phony)
	if err != nil {
		t.Fatalf("store.Add failed: %v", err)
	}
	var phonyData string
	for k, v := range remote.Data {
		phonyData = v
		delete(remote.Data, k)
	}

	orig := &chunks.Chunk{
		Type:  "testchunk",
		Level: 3,
		Buf:   []byte(GREETING),
	}
	key, err := store.Add(ctx, orig)
	if err != nil {
		t.Fatalf("store.Add failed: %v", err)
	}
	// replace it with the phony, preserving key
	for k, _ := range remote.Data {
		remote.Data[k] = phonyData
	}

	_, err = store.Get(ctx, key, "testchunk", 3)
	if err == nil {
		t.Fatalf("expected an error")
	}
	switch e := err.(type) {
	case untrusted.CorruptError:
		break
	default:
		t.Errorf("error is wrong type %T: %#v", e, e)
	}
}

func TestWrongSecret(t *testing.T) {
	remote := &kvmock.InMemory{}
	var key cas.Key

	var phonyData string
	{
		secret := &[32]byte{
			42, 42, 42, 42, 42, 42, 42, 42,
			42, 42, 42, 42, 42, 42, 42, 42,
			42, 42, 42, 42, 42, 42, 42, 42,
			42, 42, 42, 42, 42, 42, 42, 42,
		}
		converg := untrusted.New(remote, secret)
		store := kvchunks.New(converg)
		orig := &chunks.Chunk{
			Type:  "testchunk",
			Level: 3,
			Buf:   []byte(GREETING),
		}
		var err error
		ctx := context.Background()
		key, err = store.Add(ctx, orig)
		if err != nil {
			t.Fatalf("store.Add failed: %v", err)
		}
		for k, v := range remote.Data {
			phonyData = v
			delete(remote.Data, k)
		}
	}

	{
		secret := &[32]byte{
			42, 42, 42, 42, 42, 42, 42, 42,
			42, 42, 42, 42, 42, 42, 42, 42,
			42, 42, 42, 42, 42, 42, 42, 42,
			42, 42, 42, 42, 42, 42, 42, 34,
		}
		converg := untrusted.New(remote, secret)
		store := kvchunks.New(converg)

		orig := &chunks.Chunk{
			Type:  "testchunk",
			Level: 3,
			Buf:   []byte(GREETING),
		}
		var err error
		ctx := context.Background()
		key, err = store.Add(ctx, orig)
		if err != nil {
			t.Fatalf("store.Add failed: %v", err)
		}
		// replace it with the phony, preserving key
		for k, _ := range remote.Data {
			remote.Data[k] = phonyData
		}

		got, err := store.Get(ctx, key, "testchunk", 3)
		if err == nil {
			t.Fatalf("expected an error")
		}
		if got != nil {
			t.Errorf("expected no value")
		}
		switch e := err.(type) {
		case untrusted.CorruptError:
			break
		default:
			t.Errorf("error is wrong type %T: %#v", e, e)
		}
	}
}
