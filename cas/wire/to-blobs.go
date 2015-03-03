package wire

import (
	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/blobs"
)

func (m *Manifest) ToBlob(type_ string) (*blobs.Manifest, error) {
	var k cas.Key
	if err := k.UnmarshalBinary(m.Root); err != nil {
		return nil, err
	}
	manifest := &blobs.Manifest{
		Type:      type_,
		Root:      k,
		Size:      m.Size,
		ChunkSize: m.ChunkSize,
		Fanout:    m.Fanout,
	}
	return manifest, nil
}

func FromBlob(m *blobs.Manifest) *Manifest {
	return &Manifest{
		Root:      m.Root.Bytes(),
		Size:      m.Size,
		ChunkSize: m.ChunkSize,
		Fanout:    m.Fanout,
	}
}
