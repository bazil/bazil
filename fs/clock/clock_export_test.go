package clock

import "fmt"

// ValidateFile panics if internal assumptions have been broken. This
// should never happen. This method is intended for unit tests only.
//
// The assumptions here only hold true for files, not directories.
func (s *Clock) ValidateFile(parent *Clock) {
	var sync vector
	sync.merge(parent.sync)
	sync.merge(s.sync)
	if !compareLE(s.mod, sync) {
		panic(fmt.Errorf("bad internal state: %s", s))
	}
}
