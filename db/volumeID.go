package db

import (
	"encoding"
	"encoding/hex"
	"flag"
	"fmt"
)

const VolumeIDLen = 64

type VolumeID [VolumeIDLen]byte

var _ encoding.BinaryMarshaler = (*VolumeID)(nil)

func (p *VolumeID) MarshalBinary() (data []byte, err error) {
	return p[:], nil
}

var _ encoding.BinaryUnmarshaler = (*VolumeID)(nil)

func (p *VolumeID) UnmarshalBinary(data []byte) error {
	if len(data) != len(p) {
		return fmt.Errorf("volume id must be exactly %d bytes", VolumeIDLen)
	}
	copy(p[:], data)
	return nil
}

var _ flag.Value = (*VolumeID)(nil)

func (k *VolumeID) String() string {
	return hex.EncodeToString(k[:])
}

func (k *VolumeID) Set(value string) error {
	if hex.DecodedLen(len(value)) != VolumeIDLen {
		return fmt.Errorf("not a valid public key: wrong size")
	}
	if _, err := hex.Decode(k[:], []byte(value)); err != nil {
		return fmt.Errorf("not a valid public key: %v", err)
	}
	return nil
}
