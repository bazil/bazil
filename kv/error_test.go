package kv_test

import (
	"testing"

	"bazil.org/bazil/kv"
)

func TestNotFoundErrorDispay(t *testing.T) {
	k := make([]byte, 64)
	copy(k, "\x01evil\xFF")
	e := kv.NotFoundError{
		Key: k,
	}
	got := e.Error()
	if got != `Not found: 016576696cff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` {
		t.Errorf("bad error message: %q", got)
	}
}
