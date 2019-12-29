package kvmulti_test

import (
	"context"
	"reflect"
	"testing"

	"bazil.org/bazil/kv/kvmock"
	"bazil.org/bazil/kv/kvmulti"
)

func TestGetFallback(t *testing.T) {
	a := &kvmock.InMemory{}
	b := &kvmock.InMemory{}
	multi := kvmulti.New(a, b)
	ctx := context.Background()
	if err := b.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatal(err)
	}
	v, err := multi.Get(ctx, []byte("k1"))
	if err != nil {
		t.Error(err)
	}
	if g, e := string(v), "v1"; g != e {
		t.Errorf("bad value: %q != %q", g, e)
	}
}

func TestPut(t *testing.T) {
	a := &kvmock.InMemory{}
	b := &kvmock.InMemory{}
	multi := kvmulti.New(a, b)
	ctx := context.Background()
	if err := multi.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a.Data, map[string]string{"k1": "v1"}) {
		t.Errorf("bad data in a: %v", a.Data)
	}
	if !reflect.DeepEqual(b.Data, map[string]string{"k1": "v1"}) {
		t.Errorf("bad data in b: %v", a.Data)
	}
}
