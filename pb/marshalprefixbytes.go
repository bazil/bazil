package pb

import "encoding/binary"

// MarshalPrefixBytes marshals a uvarint length prefixed protobuf message.
func MarshalPrefixBytes(msg Marshaler) ([]byte, error) {
	length := msg.Size()
	buf := make([]byte, binary.MaxVarintLen64+length)
	n := binary.PutUvarint(buf, uint64(length))

	_, err := msg.MarshalTo(buf[n:])
	if err != nil {
		return nil, err
	}
	buf = buf[:n+length]
	return buf, nil
}
