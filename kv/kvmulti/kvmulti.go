package kvmulti

import (
	"errors"

	"bazil.org/bazil/kv"
)

type Multi struct {
	list []kv.KV
}

func New(k ...kv.KV) *Multi {
	return &Multi{list: k}
}

var _ = kv.KV(&Multi{})

func (m *Multi) Get(key []byte) ([]byte, error) {
	// TODO this needs to be a lot smarter
	var firstErr error
	for _, k := range m.list {
		v, err := k.Get(key)
		if err == nil {
			return v, nil
		}
		if _, isNotFoundError := err.(kv.NotFoundError); !isNotFoundError && firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return nil, firstErr
	}
	return nil, kv.NotFoundError{Key: key}
}

func (m *Multi) Put(key, value []byte) error {
	// TODO this needs to be a lot smarter
	var firstErr error
	var success bool
	for _, k := range m.list {
		err := k.Put(key, value)
		if err == nil {
			success = true
			continue
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	if !success {
		if firstErr != nil {
			return firstErr
		}
		// uhh... maybe the list was empty?
		return errors.New("weird error in kvmulti")
	}
	return nil
}
