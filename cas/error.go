package cas

import (
	"fmt"
)

// NotFound is the type of error returned by a CAS when it cannot find
// the requested key.
type NotFound struct {
	Type  string
	Level uint8
	Key   Key
}

var _ = error(NotFound{})

func (n NotFound) Error() string {
	return fmt.Sprintf("Not found: %q@%d %s", n.Type, n.Level, n.Key)
}
