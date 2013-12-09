package chunkutil

import (
	"testing"

	"bazil.org/bazil/cas"
)

func TestHashKeyEmpty(t *testing.T) {
	buf := cas.Empty.Bytes()
	k := makeKey(buf)
	if g, e := k, cas.Empty; g == e {
		t.Errorf("wrong key for falsely zero chunk: %v != %v", g, e)
	}
}

func TestHashKeyInvalid(t *testing.T) {
	buf := cas.Invalid.Bytes()
	k := makeKey(buf)
	if g, e := k, cas.Invalid; g == e {
		t.Errorf("wrong key for falsely invalid chunk: %v != %v", g, e)
	}
}
