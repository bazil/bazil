package kvmock

import "bazil.org/bazil/kv"

type InMemory struct {
	Data map[string]string
}

var _ = kv.KV(&InMemory{})

func (m *InMemory) Get(key []byte) ([]byte, error) {
	s, found := m.Data[string(key)]
	if !found {
		return nil, kv.NotFound{Key: key}
	}
	return []byte(s), nil
}

func (m *InMemory) Put(key, value []byte) error {
	if m.Data == nil {
		m.Data = make(map[string]string)
	}
	m.Data[string(key)] = string(value)
	return nil
}
