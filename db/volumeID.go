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

func (v *VolumeID) MarshalBinary() (data []byte, err error) {
	return v[:], nil
}

var _ encoding.BinaryUnmarshaler = (*VolumeID)(nil)

func (v *VolumeID) UnmarshalBinary(data []byte) error {
	if len(data) != len(v) {
		return fmt.Errorf("volume id must be exactly %d bytes", VolumeIDLen)
	}
	copy(v[:], data)
	return nil
}

var _ flag.Value = (*VolumeID)(nil)

func (v *VolumeID) String() string {
	return hex.EncodeToString(v[:])
}

func (v *VolumeID) Set(value string) error {
	if hex.DecodedLen(len(value)) != VolumeIDLen {
		return fmt.Errorf("not a valid public key: wrong size")
	}
	if _, err := hex.Decode(v[:], []byte(value)); err != nil {
		return fmt.Errorf("not a valid public key: %v", err)
	}
	return nil
}
