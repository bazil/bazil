package clock_test

import (
	"bytes"
	"testing"

	"bazil.org/bazil/fs/clock"
)

func TestMarshalBinarySimple(t *testing.T) {
	v := clock.Create(10, 3)
	v.Update(11, 2)
	got, err := v.MarshalBinary()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []byte{
		// syncLen
		2,
		// sync
		10, 3,
		11, 2,
		// modLen
		1,
		// mod
		11, 2,
		// createLen
		1,
		// create
		10, 3,
	}
	if !bytes.Equal(got, want) {
		t.Errorf("bad marshal: % x != % x", got, want)
	}
}

func TestUnmarshalBinarySimple(t *testing.T) {
	var v clock.Clock
	input := []byte{
		// syncLen
		2,
		// sync
		10, 3,
		11, 2,
		// modLen
		1,
		// mod
		11, 2,
		// createLen
		1,
		// create
		10, 3,
	}
	err := v.UnmarshalBinary(input)
	if g, e := v.String(), `{sync{10:3 11:2} mod{11:2} create{10:3}}`; g != e {
		t.Errorf("bad unmarshal: %v != %v", g, e)
	}
	if err != nil {
		t.Fatalf("got unmarshal error: %v", err)
	}
}
