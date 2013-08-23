package stash

import (
	"fmt"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/idpool"
)

// New creates a new Stash.
func New(bs chunks.Store) *Stash {
	s := &Stash{
		chunks: bs,
		local:  make(map[uint64]*chunks.Chunk),
	}
	return s
}

// Stash is a proxy for a chunks.Store, but it keeps Private Keys
// local, only saving them to the Store when Save is called.
type Stash struct {
	chunks chunks.Store
	ids    idpool.Pool
	local  map[uint64]*chunks.Chunk
}

// Get returns a chunk either from the local stash, or from the
// Store (for Private keys).
//
// For Private keys, modifying the returned chunk *will* cause the
// locally stored data to change. This is the intended usage of a
// stash.
func (s *Stash) Get(key cas.Key, typ string, level uint8) (*chunks.Chunk, error) {
	priv, ok := key.Private()
	if ok {
		chunk, ok := s.local[priv]
		if !ok {
			return nil, cas.NotFound{
				Type:  typ,
				Level: level,
				Key:   key,
			}
		}
		return chunk, nil
	}

	chunk, err := s.chunks.Get(key, typ, level)
	return chunk, err
}

func (s *Stash) drop(key cas.Key) {
	priv, ok := key.Private()
	if !ok {
		panic(fmt.Sprintf("Cannot drop non-private key: %s", key))
	}
	s.ids.Put(priv)
	delete(s.local, priv)
}

// Clone is like Get but clones the chunk if it's not already private.
// Chunks that are already private are returned as-is.
//
// A cloned chunk will have a buffer of size bytes. This is intended
// to use for re-inflating zero-trimmed chunks.
//
// Modifying the returned chunk *will* cause the locally stored data
// to change. This is the intended usage of a stash.
func (s *Stash) Clone(key cas.Key, typ string, level uint8, size uint32) (cas.Key, *chunks.Chunk, error) {
	priv, ok := key.Private()
	if ok {
		chunk, ok := s.local[priv]
		if !ok {
			return key, nil, cas.NotFound{
				Type:  typ,
				Level: level,
				Key:   key,
			}
		}
		return key, chunk, nil
	}

	chunk, err := s.Get(key, typ, level)
	if err != nil {
		return key, nil, err
	}

	// clone the byte slice
	tmp := make([]byte, size)
	copy(tmp, chunk.Buf)
	chunk.Buf = tmp

	priv = s.ids.Get()
	privkey := cas.NewKeyPrivateNum(priv)
	s.local[priv] = chunk
	return privkey, chunk, nil
}

// Save the local Chunk to the Store.
//
// On success, the old key becomes invalid.
func (s *Stash) Save(key cas.Key) (cas.Key, error) {
	priv, ok := key.Private()
	if !ok {
		return key, nil
	}

	chunk, ok := s.local[priv]
	if !ok {
		return key, cas.NotFound{
			Key: key,
		}
	}

	newkey, err := s.chunks.Add(chunk)
	if err != nil {
		return key, err
	}
	s.drop(key)
	return newkey, nil
}
