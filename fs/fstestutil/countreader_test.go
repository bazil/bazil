package fstestutil_test

import (
	"encoding/hex"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"bazil.org/bazil/fs/fstestutil"
)

const chunkSize = 4096

func TestCountReader(t *testing.T) {
	c := fstestutil.CountReader{}
	data := make([]byte, 3*chunkSize+5)
	n, err := io.ReadFull(&c, data)
	if err != nil {
		t.Fatalf("got error from read: %v", err)
	}
	if g, e := n, len(data); g != e {
		t.Errorf("wrong read length: %v != %v", g, e)
	}
	want := strings.Join([]string{
		strings.Repeat("00", chunkSize-8), "0000000000000000",
		strings.Repeat("00", chunkSize-8), "0000000000000001",
		strings.Repeat("00", chunkSize-8), "0000000000000002",
		"0000000000",
	}, "")
	if g, e := hex.EncodeToString(data), want; g != e {
		t.Errorf("wrong data: %v != %v", g, e)
	}
}

func TestCountReaderOneByte(t *testing.T) {
	c := fstestutil.CountReader{}
	data := make([]byte, 3*chunkSize+5)
	r := iotest.OneByteReader(&c)
	n, err := io.ReadFull(r, data)
	if err != nil {
		t.Fatalf("got error from read: %v", err)
	}
	if g, e := n, len(data); g != e {
		t.Errorf("wrong read length: %v != %v", g, e)
	}
	want := strings.Join([]string{
		strings.Repeat("00", chunkSize-8), "0000000000000000",
		strings.Repeat("00", chunkSize-8), "0000000000000001",
		strings.Repeat("00", chunkSize-8), "0000000000000002",
		"0000000000",
	}, "")
	if g, e := hex.EncodeToString(data), want; g != e {
		t.Errorf("wrong data: %v != %v", g, e)
	}
}
