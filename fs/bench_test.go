package fs_test

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"testing"

	bazfstestutil "bazil.org/bazil/fs/fstestutil"
	"bazil.org/bazil/util/tempdir"
)

func benchmarkWrite(b *testing.B, size int64) {
	tmp := tempdir.New(b)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(b, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(b, app)
	defer mnt.Close()

	counter := &bazfstestutil.CountReader{}
	p := path.Join(mnt.Dir, "testcontent")

	b.ResetTimer()
	b.SetBytes(size)

	for i := 0; i < b.N; i++ {
		f, err := os.Create(p)
		if err != nil {
			b.Fatalf("create: %v", err)
		}
		defer f.Close()

		_, err = io.CopyN(f, counter, size)
		if err != nil {
			b.Fatalf("write: %v", err)
		}

		err = f.Close()
		if err != nil {
			b.Fatalf("close: %v", err)
		}
	}
}

func BenchmarkWrite100(b *testing.B) {
	benchmarkWrite(b, 100)
}

func BenchmarkWrite10MB(b *testing.B) {
	benchmarkWrite(b, 10*1024*1024)
}

func BenchmarkWrite100MB(b *testing.B) {
	benchmarkWrite(b, 100*1024*1024)
}

func benchmarkRead(b *testing.B, size int64) {
	tmp := tempdir.New(b)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(b, tmp.Subdir("data"))
	defer app.Close()

	mnt := bazfstestutil.Mounted(b, app)
	defer mnt.Close()

	p := path.Join(mnt.Dir, "testcontent")

	{
		counter := &bazfstestutil.CountReader{}
		f, err := os.Create(p)
		if err != nil {
			b.Fatalf("create: %v", err)
		}
		defer f.Close()
		_, err = io.CopyN(f, counter, size)
		if err != nil {
			b.Fatalf("read: %v", err)
		}
		err = f.Close()
		if err != nil {
			b.Fatalf("close: %v", err)
		}
	}

	b.ResetTimer()
	b.SetBytes(size)

	for i := 0; i < b.N; i++ {
		f, err := os.Open(p)
		if err != nil {
			b.Fatalf("close: %v", err)
		}

		n, err := io.Copy(ioutil.Discard, f)
		if err != nil {
			b.Fatalf("read: %v", err)
		}
		if n != size {
			b.Errorf("unexpected size: %d != %d", n, size)
		}

		err = f.Close()
		if err != nil {
			b.Fatalf("close: %v", err)
		}
	}
}

func BenchmarkRead100(b *testing.B) {
	benchmarkRead(b, 100)
}

func BenchmarkRead10MB(b *testing.B) {
	benchmarkRead(b, 10*1024*1024)
}

func BenchmarkRead100MB(b *testing.B) {
	benchmarkRead(b, 100*1024*1024)
}
