package cas

import (
	"fmt"
)

// NotFoundError is the type of error returned by a CAS when it cannot
// find the requested key.
type NotFoundError struct {
	Type  string
	Level uint8
	Key   Key
}

var _ error = NotFoundError{}

func (n NotFoundError) Error() string {
	return fmt.Sprintf("Not found: %q@%d %s", n.Type, n.Level, n.Key)
}
