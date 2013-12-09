package kvchunks

// TODO maybe move this into chunks -- but then chunkutil needs to
// merge there, too, to avoid an import cycle

import (
	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/chunkutil"
	"bazil.org/bazil/kv"
)

type storeInKV struct {
	kv kv.KV
}

var _ = chunks.Store(&storeInKV{})

func makeKey(key cas.Key, typ string, level uint8) []byte {
	k := make([]byte, 0, cas.KeySize+len(typ)+1)
	k = append(k, key.Bytes()...)
	k = append(k, typ...)
	k = append(k, level)
	return k
}

func (s *storeInKV) get(key cas.Key, type_ string, level uint8) ([]byte, error) {
	k := makeKey(key, type_, level)
	data, err := s.kv.Get(k)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (s *storeInKV) Get(key cas.Key, type_ string, level uint8) (*chunks.Chunk, error) {
	return chunkutil.HandleGet(s.get, key, type_, level)
}

func (s *storeInKV) Add(chunk *chunks.Chunk) (key cas.Key, err error) {
	key = chunkutil.Hash(chunk)
	if key.IsSpecial() {
		return key, nil
	}

	k := makeKey(key, chunk.Type, chunk.Level)
	err = s.kv.Put(k, chunk.Buf)
	if err != nil {
		return cas.Invalid, err
	}
	return key, nil
}

func New(keyval kv.KV) chunks.Store {
	return &storeInKV{
		kv: keyval,
	}
}
