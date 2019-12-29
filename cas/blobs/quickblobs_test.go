package blobs_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"flag"
	"io"
	"io/ioutil"
	"math/rand"
	"testing"
	"testing/quick"

	"bazil.org/bazil/cas/blobs"
	"bazil.org/bazil/cas/chunks/mock"
	entropy "github.com/tv42/seed"
)

var seed uint64

func init() {
	// keep this as uint64 just because negative numbers are uglier and can be confused with -opt
	flag.Uint64Var(&seed, "seed", 0, "seed to initialize random number generator")
}

type randReader struct {
	*rand.Rand
}

func (r randReader) Read(p []byte) (n int, err error) {
	for len(p) > 4 {
		binary.BigEndian.PutUint32(p, r.Uint32())
		n += 4
		p = p[4:]
	}
	if len(p) > 0 {
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, r.Uint32())
		n += copy(p, buf)
	}
	return n, err
}

func NewRandReader(seed int64) randReader {
	src := rand.NewSource(seed)
	rnd := rand.New(src)
	return randReader{rnd}
}

func TestCompareRead(t *testing.T) {
	r := NewRandReader(42)
	buf := make([]byte, 10*1024*1024)
	r.Read(buf)

	blob := emptyBlob(t, &mock.InMemory{})
	ctx := context.Background()
	blob.IO(ctx).WriteAt(buf, 0)

	got := func(p []byte, off int64) (int, error) {
		if off < 0 {
			off = -off
		}
		return blob.IO(ctx).ReadAt(p, off)
	}
	rat := bytes.NewReader(buf)
	exp := func(p []byte, off int64) (int, error) {
		if off < 0 {
			off = -off
		}
		return rat.ReadAt(p, off)
	}

	config := quick.Config{
		MaxCountScale: 100.0,
	}
	if err := quick.CheckEqual(got, exp, &config); err != nil {
		t.Error(err)
	}
}

func testCompareBoth(t *testing.T, saveEvery int) {
	f, err := ioutil.TempFile("", "baziltest-")
	if err != nil {
		t.Fatalf("tempfile error: %v", err)
	}
	defer f.Close()

	blob, err := blobs.Open(&mock.InMemory{},
		&blobs.Manifest{
			Type:      "footype",
			ChunkSize: blobs.MinChunkSize,
			Fanout:    2,
		},
	)
	if err != nil {
		t.Fatalf("cannot open blob: %v", err)
	}

	if seed == 0 {
		seed = uint64(entropy.Seed())
	}
	t.Logf("Seed is %d", seed)
	qconf := quick.Config{
		Rand: rand.New(rand.NewSource(int64(seed))),
	}

	ctx := context.Background()
	count := 0
	got := func(isWrite bool, off int64, size int, writeSeed int64) (num int, read []byte, err error) {
		if off < 0 {
			off = -off
		}
		off = off % (10 * 1024 * 1024)

		if size < 0 {
			size = -size
		}
		size = size % (10 * 1024)

		if isWrite {
			count++
			if saveEvery > 0 && count%saveEvery == 0 {
				_, err := blob.Save(ctx)
				if err != nil {
					return 0, nil, err
				}
			}

			p := make([]byte, size)
			NewRandReader(writeSeed).Read(p)
			t.Logf("write %d@%d", len(p), off)
			n, err := blob.IO(ctx).WriteAt(p, off)
			return n, nil, err
		} else {
			p := make([]byte, size)
			t.Logf("read %d@%d", len(p), off)
			n, err := blob.IO(ctx).ReadAt(p, off)

			// http://golang.org/pkg/io/#ReaderAt says "If the n = len(p)
			// bytes returned by ReadAt are at the end of the input
			// source, ReadAt may return either err == EOF or err ==
			// nil." Unify the result
			if n == len(p) && err == io.EOF {
				err = nil
			}

			return n, p, err
		}
	}

	exp := func(isWrite bool, off int64, size int, writeSeed int64) (num int, read []byte, err error) {
		if off < 0 {
			off = -off
		}
		off = off % (10 * 1024 * 1024)

		if size < 0 {
			size = -size
		}
		size = size % (10 * 1024)

		if isWrite {
			p := make([]byte, size)
			NewRandReader(writeSeed).Read(p)
			n, err := f.WriteAt(p, off)
			return n, nil, err
		} else {
			p := make([]byte, size)
			n, err := f.ReadAt(p, off)

			// http://golang.org/pkg/io/#ReaderAt says "If the n = len(p)
			// bytes returned by ReadAt are at the end of the input
			// source, ReadAt may return either err == EOF or err ==
			// nil." Unify the result
			if n == len(p) && err == io.EOF {
				err = nil
			}

			return n, p, err
		}
	}

	if err := quick.CheckEqual(got, exp, &qconf); err != nil {
		t.Error(err)
	}
}

func TestCompareBothNoSave(t *testing.T) {
	testCompareBoth(t, 0)
}

func TestCompareBoth10(t *testing.T) {
	testCompareBoth(t, 10)
}
