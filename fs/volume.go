package fs

const VolumeIDLen = 64

type VolumeID [VolumeIDLen]byte

func (id *VolumeID) Bytes() []byte {
	return id[:]
}
