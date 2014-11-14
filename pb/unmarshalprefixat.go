package pb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/gogo/protobuf/proto"
)

// UnmarshalPrefixAt unmarshals a uvarint length prefixed protobuf
// message.
func UnmarshalPrefixAt(rat io.ReaderAt, off int64, msg proto.Unmarshaler) (n int, err error) {
	var length uint64
	{
		var buf [binary.MaxVarintLen64]byte
		var varlen int
		varlen, err = rat.ReadAt(buf[:], off)
		switch {
		case err == io.EOF && varlen > 0:
			// ignore EOF here if we got at least something
		case err != nil:
			return n, err
		}
		length, n = binary.Uvarint(buf[:varlen])
		if n <= 0 {
			return -n, errors.New("length header is corrupt")
		}
	}

	// signal zero message to caller, so they can ignore
	if length == 0 {
		return n, EmptyMessage
	}

	buf := make([]byte, length)
	n2, err := rat.ReadAt(buf, off+int64(n))
	n += n2
	switch {
	case err == io.EOF && uint64(n2) == length:
		// ignore EOF if we got all we needed
	case err != nil:
		return n, err
	}
	err = msg.Unmarshal(buf)
	if err != nil {
		return n, fmt.Errorf("unmarshal problem: %v", err)
	}
	return n, nil
}
