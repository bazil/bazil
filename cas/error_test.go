package cas_test

import (
	"testing"

	"bazil.org/bazil/cas"
)

func TestNotFoundErrorDispay(t *testing.T) {
	k := make([]byte, cas.KeySize)
	copy(k, "\x01evil\xFF")
	e := cas.NotFoundError{
		Type:  "blob",
		Level: 4,
		Key:   cas.NewKey(k),
	}
	got := e.Error()
	if got != `Not found: "blob"@4 016576696cff00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000` {
		t.Errorf("bad error message: %q", got)
	}
}
