package kv

type KV interface {
	Get(key []byte) ([]byte, error)
	Put(key, value []byte) error
}
