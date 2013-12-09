package chunkutil

import (
	"bazil.org/bazil/cas/chunks"
)

// MakeChunk makes a new Chunk of the given description, filling it
// with content from data.
func MakeChunk(typ string, level uint8, data []byte) *chunks.Chunk {
	chunk := &chunks.Chunk{
		Type:  typ,
		Level: level,
		Buf:   data,
	}
	return chunk
}
