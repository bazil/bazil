package wire

import (
	"bazil.org/bazil/cas/blobs"
)

func (m *Manifest) ToBlob(type_ string) *blobs.Manifest {
	return &blobs.Manifest{
		Type:      type_,
		Root:      m.Root,
		Size:      m.Size,
		ChunkSize: m.ChunkSize,
		Fanout:    m.Fanout,
	}
}

func FromBlob(m *blobs.Manifest) Manifest {
	return Manifest{
		Root:      m.Root,
		Size:      m.Size,
		ChunkSize: m.ChunkSize,
		Fanout:    m.Fanout,
	}
}
