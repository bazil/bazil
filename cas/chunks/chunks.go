// Package chunks implements low-level storage for chunks of data.
package chunks

import (
	"context"
	"fmt"

	"bazil.org/bazil/cas"
)

// Chunk is a chunk of data, to be stored in a CAS. A Chunk is assumed
// to be small enough to fit fully in memory in a single contiguous
// range of bytes.
type Chunk struct {
	Type  string
	Level uint8
	Buf   []byte
}

func (c *Chunk) String() string {
	return fmt.Sprintf("Chunk{%q@%d %x}", c.Type, c.Level, c.Buf)
}

// Store is a low-level CAS store that stores limited size chunks. It
// is not meant to store files directly.
type Store interface {
	// Get a chunk from the chunk store.
	//
	// The returned Chunk is considered read-only and must not be
	// modified.
	Get(ctx context.Context, key cas.Key, type_ string, level uint8) (*Chunk, error)

	// Add a chunk to the chunk store.
	Add(ctx context.Context, chunk *Chunk) (key cas.Key, err error)
}
