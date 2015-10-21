package chunkutil

import (
	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"golang.org/x/net/context"
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

type Handler func(ctx context.Context, key cas.Key, typ string, level uint8) ([]byte, error)

func HandleGet(ctx context.Context, fn Handler, key cas.Key, typ string, level uint8) (*chunks.Chunk, error) {
	if key.IsSpecial() {
		if key == cas.Empty {
			chunk := MakeChunk(typ, level, nil)
			return chunk, nil
		}
		return nil, cas.NotFoundError{
			Type:  typ,
			Level: level,
			Key:   key,
		}
	}

	data, err := fn(ctx, key, typ, level)
	if err != nil {
		return nil, err
	}

	if data == nil {
		return nil, cas.NotFoundError{
			Type:  typ,
			Level: level,
			Key:   key,
		}
	}

	chunk := MakeChunk(typ, level, data)
	return chunk, nil

}
