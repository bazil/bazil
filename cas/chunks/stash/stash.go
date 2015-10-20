package stash

import (
	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/idpool"
	"golang.org/x/net/context"
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
func (s *Stash) Get(ctx context.Context, key cas.Key, typ string, level uint8) (*chunks.Chunk, error) {
	priv, ok := key.Private()
	if ok {
		chunk, ok := s.local[priv]
		if !ok {
			return nil, cas.NotFoundError{
				Type:  typ,
				Level: level,
				Key:   key,
			}
		}
		return chunk, nil
	}

	chunk, err := s.chunks.Get(ctx, key, typ, level)
	return chunk, err
}

func (s *Stash) drop(priv uint64) {
	s.ids.Put(priv)
	delete(s.local, priv)
}

// Drop forgets a Private chunk. The key may be reused, so caller must
// not remember the old key.
func (s *Stash) Drop(key cas.Key) {
	priv, ok := key.Private()
	if !ok {
		return
	}
	s.drop(priv)
}

// Clone is like Get but clones the chunk if it's not already private.
// Chunks that are already private are returned as-is.
//
// A cloned chunk will have a buffer of size bytes. This is intended
// to use for re-inflating zero-trimmed chunks.
//
// Modifying the returned chunk *will* cause the locally stored data
// to change. This is the intended usage of a stash.
func (s *Stash) Clone(ctx context.Context, key cas.Key, typ string, level uint8, size uint32) (cas.Key, *chunks.Chunk, error) {
	priv, ok := key.Private()
	if ok {
		chunk, ok := s.local[priv]
		if !ok {
			return key, nil, cas.NotFoundError{
				Type:  typ,
				Level: level,
				Key:   key,
			}
		}
		return key, chunk, nil
	}

	chunk, err := s.chunks.Get(ctx, key, typ, level)
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
func (s *Stash) Save(ctx context.Context, key cas.Key) (cas.Key, error) {
	priv, ok := key.Private()
	if !ok {
		return key, nil
	}

	chunk, ok := s.local[priv]
	if !ok {
		return key, cas.NotFoundError{
			Key: key,
		}
	}

	newkey, err := s.chunks.Add(ctx, chunk)
	if err != nil {
		return key, err
	}
	s.drop(priv)
	return newkey, nil
}

// Clear drops all the Private chunks held in this Stash. This is
// useful e.g. when the contents of a Blob are completely rewritten.
func (s *Stash) Clear() {
	s.ids = idpool.Pool{}
	s.local = make(map[uint64]*chunks.Chunk)
}
