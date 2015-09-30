package clock

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"errors"
	"math"
)

// The marshaled format is
//
//     <syncEntries uvarint><sync vector>
//     <modEntries uvarint><mod vector>
//     <createEntries uvarint><create vector>
//
// where vector is a sequence of
//
//     <id uint32:uvarint><t uint64:uvarint>
//
// sorted by id.
//
// The create vector is always of length 0 (tombstone) or 1 (normal
// case).

var _ encoding.BinaryMarshaler = (*Clock)(nil)
var _ encoding.BinaryUnmarshaler = (*Clock)(nil)

func marshalUvarint(buf *bytes.Buffer, n uint64) {
	tmp := make([]byte, binary.MaxVarintLen64)
	size := binary.PutUvarint(tmp, n)
	_, _ = buf.Write(tmp[:size])
}

func marshalVector(buf *bytes.Buffer, vec vector) {
	marshalUvarint(buf, uint64(len(vec.list)))
	for _, it := range vec.list {
		marshalUvarint(buf, uint64(it.id))
		marshalUvarint(buf, uint64(it.t))
	}
}

// MarshalBinary encodes Version into binary form.
func (v *Clock) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer
	marshalVector(&buf, v.sync)
	marshalVector(&buf, v.mod)
	marshalVector(&buf, v.create)
	return buf.Bytes(), nil
}

func unmarshalItem(buf *bytes.Reader, it *item) error {
	var tmp uint64
	var err error

	if tmp, err = binary.ReadUvarint(buf); err != nil {
		return err
	}
	if tmp > uint64(MaxPeer) {
		return errors.New("vector item id beyond peer id space")
	}
	it.id = Peer(tmp)

	t, err := binary.ReadUvarint(buf)
	if err != nil {
		return err
	}
	it.t = Epoch(t)
	return nil
}

func unmarshalVector(buf *bytes.Reader, vec *vector) error {
	var num uint64
	var err error
	if num, err = binary.ReadUvarint(buf); err != nil {
		return err
	}
	if num > math.MaxUint32 {
		return errors.New("vector length impossibly high")
	}

	vec.list = make([]item, num)
	for i := uint32(0); i < uint32(num); i++ {
		if err := unmarshalItem(buf, &vec.list[i]); err != nil {
			return err
		}
	}

	return nil
}

// UnmarshalBinary decodes Version from binary form.
func (v *Clock) UnmarshalBinary(p []byte) error {
	buf := bytes.NewReader(p)
	if err := unmarshalVector(buf, &v.sync); err != nil {
		return err
	}
	if err := unmarshalVector(buf, &v.mod); err != nil {
		return err
	}
	if err := unmarshalVector(buf, &v.create); err != nil {
		return err
	}
	if buf.Len() > 0 {
		return errors.New("too much data to unmarshal")
	}
	return nil
}
