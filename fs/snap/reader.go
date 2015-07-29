package snap

import (
	"io"
	"os"

	"bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/pb"
)

type Reader struct {
	rat   io.ReaderAt
	align int64
}

func NewReader(rat io.ReaderAt, align uint32) (*Reader, error) {
	reader := &Reader{
		rat:   rat,
		align: int64(align),
	}
	return reader, nil
}

// TODO how to pass ctx to rat.ReadAt
func (r *Reader) Lookup(name string) (*wire.Dirent, error) {
	// TODO binary search
	it := r.Iter()
	var de *wire.Dirent
	var err error
	for {
		de, err = it.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if de.Name == name {
			return de, nil
		}
	}
	return nil, os.ErrNotExist
}

func (r *Reader) Iter() *Iterator {
	return &Iterator{r: r}
}

type Iterator struct {
	r   *Reader
	off int64
}

func (i *Iterator) Next() (*wire.Dirent, error) {
	var de wire.Dirent
	for {
		n, err := pb.UnmarshalPrefixAt(i.r.rat, i.off, &de)
		if err == pb.ErrEmptyMessage {
			i.off += int64(n)
			continue
		}
		if err != nil {
			return nil, err
		}
		i.off += int64(n)
		return &de, nil
	}
}
