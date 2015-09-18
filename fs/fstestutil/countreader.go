package fstestutil

import (
	"encoding/binary"
	"io"
)

type CountReader struct {
	offset uint64
}

const chunkSize = 4096
const counterSize = 8
const counterAlign = chunkSize - counterSize

var zeroes [chunkSize]byte

func (c *CountReader) Read(p []byte) (n int, err error) {
	for len(p) > 0 {
		offset := c.offset % chunkSize
		if offset < counterAlign {
			// distance to next counter location
			skip := counterAlign - offset
			nn := uint64(copy(p, zeroes[:skip]))
			c.offset += nn
			offset += nn
			n += int(nn)
			p = p[nn:]
		}
		if offset >= chunkSize {
			continue
		}
		if offset >= counterAlign {
			align := offset - counterAlign
			var num [counterSize]byte
			binary.BigEndian.PutUint64(num[:], c.offset/chunkSize)
			nn := uint64(copy(p, num[align:]))
			c.offset += nn
			n += int(nn)
			p = p[nn:]
		}
	}
	return n, nil
}

var _ io.Reader = (*CountReader)(nil)
