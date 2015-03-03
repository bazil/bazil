package pb

import (
	"encoding/binary"

	"github.com/golang/protobuf/proto"
)

// MarshalPrefixBytes marshals a uvarint length prefixed protobuf message.
func MarshalPrefixBytes(msg proto.Message) ([]byte, error) {
	m, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}

	buf := make([]byte, binary.MaxVarintLen64+len(m))
	n := binary.PutUvarint(buf, uint64(len(m)))
	buf = buf[:n+len(m)]
	copy(buf[n:], m)
	return buf, nil
}
