package kvmock

import (
	"bazil.org/bazil/kv"
	"golang.org/x/net/context"
)

type InMemory struct {
	Data map[string]string
}

var _ kv.KV = (*InMemory)(nil)

func (m *InMemory) Get(ctx context.Context, key []byte) ([]byte, error) {
	s, found := m.Data[string(key)]
	if !found {
		return nil, kv.NotFoundError{Key: key}
	}
	return []byte(s), nil
}

func (m *InMemory) Put(ctx context.Context, key, value []byte) error {
	if m.Data == nil {
		m.Data = make(map[string]string)
	}
	m.Data[string(key)] = string(value)
	return nil
}
