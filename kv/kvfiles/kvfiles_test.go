package kvfiles_test

import (
	"testing"

	"bazil.org/bazil/kv/kvfiles"
	"bazil.org/bazil/util/tempdir"
)

func TestAdd(t *testing.T) {
	temp := tempdir.New(t)
	defer temp.Cleanup()

	k, err := kvfiles.Open(temp.Path)
	if err != nil {
		t.Fatalf("kvfiles.Open fail: %v\n", err)
	}

	err = k.Put([]byte("quux"), []byte("foobar"))
	if err != nil {
		t.Fatalf("c.Put fail: %v\n", err)
	}
}

func TestGet(t *testing.T) {
	temp := tempdir.New(t)
	defer temp.Cleanup()

	c, err := kvfiles.Open(temp.Path)
	if err != nil {
		t.Fatalf("kvfiles.Open fail: %v\n", err)
	}

	err = c.Put([]byte("quux"), []byte("foobar"))
	if err != nil {
		t.Fatalf("c.Put fail: %v\n", err)
	}

	data, err := c.Get([]byte("quux"))
	if err != nil {
		t.Fatalf("c.Get failed: %v", err)
	}
	if g, e := string(data), "foobar"; g != e {
		t.Fatalf("c.Get gave wrong content: %q != %q", g, e)
	}
}

func TestPutOverwrite(t *testing.T) {
	temp := tempdir.New(t)
	defer temp.Cleanup()

	k, err := kvfiles.Open(temp.Path)
	if err != nil {
		t.Fatalf("kvfiles.Open fail: %v\n", err)
	}

	err = k.Put([]byte("quux"), []byte("foobar"))
	if err != nil {
		t.Fatalf("k.Put fail: %v\n", err)
	}

	err = k.Put([]byte("quux"), []byte("foobar"))
	if err != nil {
		t.Fatalf("k.Put fail: %v\n", err)
	}
}
