// Code generated by protoc-gen-gogo.
// source: bazil.org/bazil/fs/wire/volume.proto
// DO NOT EDIT!

package wire

import proto "code.google.com/p/gogoprotobuf/proto"
import json "encoding/json"
import math "math"

// discarding unused import gogoproto "code.google.com/p/gogoprotobuf/gogoproto/gogo.pb"

import io3 "io"
import code_google_com_p_gogoprotobuf_proto3 "code.google.com/p/gogoprotobuf/proto"

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type KV struct {
	Local            *KV_Local      `protobuf:"bytes,1,opt,name=local" json:"local,omitempty"`
	External         []*KV_External `protobuf:"bytes,2,rep,name=external" json:"external,omitempty"`
	XXX_unrecognized []byte         `json:"-"`
}

func (m *KV) Reset()         { *m = KV{} }
func (m *KV) String() string { return proto.CompactTextString(m) }
func (*KV) ProtoMessage()    {}

func (m *KV) GetLocal() *KV_Local {
	if m != nil {
		return m.Local
	}
	return nil
}

func (m *KV) GetExternal() []*KV_External {
	if m != nil {
		return m.External
	}
	return nil
}

type KV_Local struct {
	Secret           []byte `protobuf:"bytes,1,opt,name=secret" json:"secret"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *KV_Local) Reset()         { *m = KV_Local{} }
func (m *KV_Local) String() string { return proto.CompactTextString(m) }
func (*KV_Local) ProtoMessage()    {}

func (m *KV_Local) GetSecret() []byte {
	if m != nil {
		return m.Secret
	}
	return nil
}

type KV_External struct {
	Path             string `protobuf:"bytes,1,req,name=path" json:"path"`
	Secret           []byte `protobuf:"bytes,2,opt,name=secret" json:"secret"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *KV_External) Reset()         { *m = KV_External{} }
func (m *KV_External) String() string { return proto.CompactTextString(m) }
func (*KV_External) ProtoMessage()    {}

func (m *KV_External) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

func (m *KV_External) GetSecret() []byte {
	if m != nil {
		return m.Secret
	}
	return nil
}

type VolumeConfig struct {
	VolumeID         []byte `protobuf:"bytes,1,req,name=volumeID" json:"volumeID"`
	Storage          KV     `protobuf:"bytes,2,req,name=storage" json:"storage"`
	XXX_unrecognized []byte `json:"-"`
}

func (m *VolumeConfig) Reset()         { *m = VolumeConfig{} }
func (m *VolumeConfig) String() string { return proto.CompactTextString(m) }
func (*VolumeConfig) ProtoMessage()    {}

func (m *VolumeConfig) GetVolumeID() []byte {
	if m != nil {
		return m.VolumeID
	}
	return nil
}

func (m *VolumeConfig) GetStorage() KV {
	if m != nil {
		return m.Storage
	}
	return KV{}
}

func init() {
}
func (m *KV) Unmarshal(data []byte) error {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return io3.ErrUnexpectedEOF
			}
			b := data[index]
			index++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + msglen
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			if m.Local == nil {
				m.Local = &KV_Local{}
			}
			if err := m.Local.Unmarshal(data[index:postIndex]); err != nil {
				return err
			}
			index = postIndex
		case 2:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + msglen
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			m.External = append(m.External, &KV_External{})
			m.External[len(m.External)-1].Unmarshal(data[index:postIndex])
			index = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			index -= sizeOfWire
			skippy, err := code_google_com_p_gogoprotobuf_proto3.Skip(data[index:])
			if err != nil {
				return err
			}
			if (index + skippy) > l {
				return io3.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[index:index+skippy]...)
			index += skippy
		}
	}
	return nil
}
func (m *KV_Local) Unmarshal(data []byte) error {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return io3.ErrUnexpectedEOF
			}
			b := data[index]
			index++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + byteLen
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			m.Secret = append(m.Secret, data[index:postIndex]...)
			index = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			index -= sizeOfWire
			skippy, err := code_google_com_p_gogoprotobuf_proto3.Skip(data[index:])
			if err != nil {
				return err
			}
			if (index + skippy) > l {
				return io3.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[index:index+skippy]...)
			index += skippy
		}
	}
	return nil
}
func (m *KV_External) Unmarshal(data []byte) error {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return io3.ErrUnexpectedEOF
			}
			b := data[index]
			index++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				stringLen |= (uint64(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + int(stringLen)
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			m.Path = string(data[index:postIndex])
			index = postIndex
		case 2:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + byteLen
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			m.Secret = append(m.Secret, data[index:postIndex]...)
			index = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			index -= sizeOfWire
			skippy, err := code_google_com_p_gogoprotobuf_proto3.Skip(data[index:])
			if err != nil {
				return err
			}
			if (index + skippy) > l {
				return io3.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[index:index+skippy]...)
			index += skippy
		}
	}
	return nil
}
func (m *VolumeConfig) Unmarshal(data []byte) error {
	l := len(data)
	index := 0
	for index < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if index >= l {
				return io3.ErrUnexpectedEOF
			}
			b := data[index]
			index++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var byteLen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				byteLen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + byteLen
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			m.VolumeID = append(m.VolumeID, data[index:postIndex]...)
			index = postIndex
		case 2:
			if wireType != 2 {
				return code_google_com_p_gogoprotobuf_proto3.ErrWrongType
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if index >= l {
					return io3.ErrUnexpectedEOF
				}
				b := data[index]
				index++
				msglen |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			postIndex := index + msglen
			if postIndex > l {
				return io3.ErrUnexpectedEOF
			}
			if err := m.Storage.Unmarshal(data[index:postIndex]); err != nil {
				return err
			}
			index = postIndex
		default:
			var sizeOfWire int
			for {
				sizeOfWire++
				wire >>= 7
				if wire == 0 {
					break
				}
			}
			index -= sizeOfWire
			skippy, err := code_google_com_p_gogoprotobuf_proto3.Skip(data[index:])
			if err != nil {
				return err
			}
			if (index + skippy) > l {
				return io3.ErrUnexpectedEOF
			}
			m.XXX_unrecognized = append(m.XXX_unrecognized, data[index:index+skippy]...)
			index += skippy
		}
	}
	return nil
}
func (m *KV) Size() (n int) {
	var l int
	_ = l
	if m.Local != nil {
		l = m.Local.Size()
		n += 1 + l + sovVolume(uint64(l))
	}
	if len(m.External) > 0 {
		for _, e := range m.External {
			l = e.Size()
			n += 1 + l + sovVolume(uint64(l))
		}
	}
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}
func (m *KV_Local) Size() (n int) {
	var l int
	_ = l
	l = len(m.Secret)
	n += 1 + l + sovVolume(uint64(l))
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}
func (m *KV_External) Size() (n int) {
	var l int
	_ = l
	l = len(m.Path)
	n += 1 + l + sovVolume(uint64(l))
	l = len(m.Secret)
	n += 1 + l + sovVolume(uint64(l))
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}
func (m *VolumeConfig) Size() (n int) {
	var l int
	_ = l
	l = len(m.VolumeID)
	n += 1 + l + sovVolume(uint64(l))
	l = m.Storage.Size()
	n += 1 + l + sovVolume(uint64(l))
	if m.XXX_unrecognized != nil {
		n += len(m.XXX_unrecognized)
	}
	return n
}

func sovVolume(x uint64) (n int) {
	for {
		n++
		x >>= 7
		if x == 0 {
			break
		}
	}
	return n
}
func sozVolume(x uint64) (n int) {
	return sovVolume(uint64((x << 1) ^ uint64((int64(x) >> 63))))
	return sovVolume(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *KV) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *KV) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	if m.Local != nil {
		data[i] = 0xa
		i++
		i = encodeVarintVolume(data, i, uint64(m.Local.Size()))
		n1, err := m.Local.MarshalTo(data[i:])
		if err != nil {
			return 0, err
		}
		i += n1
	}
	if len(m.External) > 0 {
		for _, msg := range m.External {
			data[i] = 0x12
			i++
			i = encodeVarintVolume(data, i, uint64(msg.Size()))
			n, err := msg.MarshalTo(data[i:])
			if err != nil {
				return 0, err
			}
			i += n
		}
	}
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}
func (m *KV_Local) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *KV_Local) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintVolume(data, i, uint64(len(m.Secret)))
	i += copy(data[i:], m.Secret)
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}
func (m *KV_External) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *KV_External) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintVolume(data, i, uint64(len(m.Path)))
	i += copy(data[i:], m.Path)
	data[i] = 0x12
	i++
	i = encodeVarintVolume(data, i, uint64(len(m.Secret)))
	i += copy(data[i:], m.Secret)
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}
func (m *VolumeConfig) Marshal() (data []byte, err error) {
	size := m.Size()
	data = make([]byte, size)
	n, err := m.MarshalTo(data)
	if err != nil {
		return nil, err
	}
	return data[:n], nil
}

func (m *VolumeConfig) MarshalTo(data []byte) (n int, err error) {
	var i int
	_ = i
	var l int
	_ = l
	data[i] = 0xa
	i++
	i = encodeVarintVolume(data, i, uint64(len(m.VolumeID)))
	i += copy(data[i:], m.VolumeID)
	data[i] = 0x12
	i++
	i = encodeVarintVolume(data, i, uint64(m.Storage.Size()))
	n2, err := m.Storage.MarshalTo(data[i:])
	if err != nil {
		return 0, err
	}
	i += n2
	if m.XXX_unrecognized != nil {
		i += copy(data[i:], m.XXX_unrecognized)
	}
	return i, nil
}
func encodeFixed64Volume(data []byte, offset int, v uint64) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	data[offset+4] = uint8(v >> 32)
	data[offset+5] = uint8(v >> 40)
	data[offset+6] = uint8(v >> 48)
	data[offset+7] = uint8(v >> 56)
	return offset + 8
}
func encodeFixed32Volume(data []byte, offset int, v uint32) int {
	data[offset] = uint8(v)
	data[offset+1] = uint8(v >> 8)
	data[offset+2] = uint8(v >> 16)
	data[offset+3] = uint8(v >> 24)
	return offset + 4
}
func encodeVarintVolume(data []byte, offset int, v uint64) int {
	for v >= 1<<7 {
		data[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	data[offset] = uint8(v)
	return offset + 1
}
