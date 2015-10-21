package kv

import (
	"golang.org/x/net/context"
)

type KV interface {
	Get(ctx context.Context, key []byte) ([]byte, error)
	Put(ctx context.Context, key, value []byte) error
}
