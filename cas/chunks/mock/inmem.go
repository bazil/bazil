package mock

import (
	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/chunkutil"
)

type mapkey struct {
	Key   cas.Key
	Type  string
	Level uint8
}

// InMemory is a chunks.Store that all chunks in an in-memory map.
// It is intended for unit test use only.
type InMemory struct {
	m map[mapkey][]byte
}

var _ chunks.Store = (*InMemory)(nil)

func (c *InMemory) get(key cas.Key, typ string, level uint8) ([]byte, error) {
	data := c.m[mapkey{key, typ, level}]
	return data, nil
}

// Get fetches a Chunk. See chunks.Store.Get.
func (c *InMemory) Get(key cas.Key, typ string, level uint8) (*chunks.Chunk, error) {
	return chunkutil.HandleGet(c.get, key, typ, level)
}

// Add adds a Chunk to the Store. See chunks.Store.Add.
func (c *InMemory) Add(chunk *chunks.Chunk) (key cas.Key, err error) {
	key = chunkutil.Hash(chunk)
	if c.m == nil {
		c.m = make(map[mapkey][]byte)
	}
	c.m[mapkey{key, chunk.Type, chunk.Level}] = chunk.Buf
	return key, nil
}
