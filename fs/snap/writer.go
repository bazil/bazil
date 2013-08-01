package snap

import (
	"errors"
	"io"
	"math"

	"bazil.org/bazil/fs/snap/wire"
	"bazil.org/bazil/pb"
)

type Writer struct {
	wat   io.WriterAt
	align int64
	off   int64
	err   error
}

func NewWriter(wat io.WriterAt) *Writer {
	writer := &Writer{wat: wat}
	writer.align = 4096
	return writer
}

func (s *Writer) Align() uint32 {
	// guaranteed to fit by .Add checking after growing
	return uint32(s.align)
}

func (s *Writer) Add(de *wire.Dirent) error {
	if s.err != nil {
		return s.err
	}
	var buf []byte
	buf, s.err = pb.MarshalPrefixBytes(de)
	if s.err != nil {
		return s.err
	}

	// make sure we fit within the next align-sized block
	if int64(len(buf)) > s.align-(s.off%s.align) {
		// would not fit

		for int64(len(buf)) > s.align {
			// would never fit with current value of align

			// because we always double it, old alignments are all valid;
			// there may just be some unnecessary padding inside the
			// new-size blocks
			newAlign := s.align * 2
			if newAlign > math.MaxUint32 {
				s.err = errors.New("dirent alignment grew too big")
				return s.err
			}
			s.align = newAlign
		}

		// go to next aligned boundary after s.off
		//
		// https://en.wikipedia.org/wiki/Data_structure_alignment#Computing_padding
		s.off = (s.off + s.align - 1) & ^(s.align - 1)
	}

	var n int
	n, s.err = s.wat.WriteAt(buf, s.off)
	s.off += int64(n)
	return s.err
}
