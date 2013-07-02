package kv

import (
	"fmt"
)

// NotFound is the type of error returned by a KV when it cannot find
// the requested key.
type NotFound struct {
	Key []byte
}

var _ = error(NotFound{})

func (n NotFound) Error() string {
	return fmt.Sprintf("Not found: %x", n.Key)
}
