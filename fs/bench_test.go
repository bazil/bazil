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

func benchmark(b *testing.B, fn func(b *testing.B, mnt string)) {
	tmp := tempdir.New(b)
	defer tmp.Cleanup()
	app := bazfstestutil.NewApp(b, tmp.Subdir("data"))
	defer app.Close()
	bazfstestutil.CreateVolume(b, app, "default")

	mnt := bazfstestutil.Mounted(b, app, "default")
	defer mnt.Close()

	fn(b, mnt.Dir)
}

func doWrites(size int64) func(b *testing.B, mnt string) {
	return func(b *testing.B, mnt string) {
		counter := &bazfstestutil.CountReader{}
		p := path.Join(mnt, "testcontent")

		b.SetBytes(size)
		b.ResetTimer()

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
}

func BenchmarkWrite100(b *testing.B) {
	benchmark(b, doWrites(100))
}

func BenchmarkWrite10MB(b *testing.B) {
	benchmark(b, doWrites(10*1024*1024))
}

func BenchmarkWrite100MB(b *testing.B) {
	benchmark(b, doWrites(100*1024*1024))
}

func doReads(size int64) func(b *testing.B, mnt string) {
	return func(b *testing.B, mnt string) {
		p := path.Join(mnt, "testcontent")

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

		b.SetBytes(size)
		b.ResetTimer()

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
}

func BenchmarkRead100(b *testing.B) {
	benchmark(b, doReads(100))
}

func BenchmarkRead10MB(b *testing.B) {
	benchmark(b, doReads(10*1024*1024))
}

func BenchmarkRead100MB(b *testing.B) {
	benchmark(b, doReads(100*1024*1024))
}
