package kv

import (
	"fmt"
)

// NotFoundError is the type of error returned by a KV when it cannot
// find the requested key.
type NotFoundError struct {
	Key []byte
}

var _ = error(NotFoundError{})

func (n NotFoundError) Error() string {
	return fmt.Sprintf("Not found: %x", n.Key)
}
