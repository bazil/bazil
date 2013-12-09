package chunkutil_test

import (
	"testing"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/chunkutil"
)

func TestHashEmpty(t *testing.T) {
	chunk := &chunks.Chunk{
		Type:  "testchunk",
		Level: 42,
		Buf:   []byte{},
	}
	k := chunkutil.Hash(chunk)
	if g, e := k, cas.Empty; g != e {
		t.Errorf("wrong key for zero chunk: %v != %v", g, e)
	}
}

func TestHashSomeZeroes(t *testing.T) {
	chunk := &chunks.Chunk{
		Type:  "testchunk",
		Level: 42,
		Buf:   []byte{0x00, 0x00, 0x00},
	}
	k := chunkutil.Hash(chunk)
	if g, e := k.String(), "85edc8cb5137ca38de1351f85216b2c59e4f19bcd334bb9fca26a68dcf606a1a7c9a9c5c445522e068336f98f1729d32a057b6f96dbdc18158558d5b4d518860"; g != e {
		t.Errorf("wrong key for some zero bytes: %v != %v", g, e)
	}
}
