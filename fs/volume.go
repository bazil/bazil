package fs

import (
	"crypto/rand"
	"errors"
)

const VolumeIDLen = 64

type VolumeID [VolumeIDLen]byte

func (id *VolumeID) Bytes() []byte {
	return id[:]
}

func NewVolumeID(b []byte) (*VolumeID, error) {
	var v VolumeID
	n := copy(v[:], b)
	if n != VolumeIDLen {
		return nil, errors.New("invalid volume id length")
	}
	return &v, nil
}

func RandomVolumeID() (*VolumeID, error) {
	var id VolumeID
	_, err := rand.Read(id[:])
	if err != nil {
		return nil, err
	}
	return &id, nil
}
