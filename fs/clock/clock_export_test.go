package clock

import "fmt"

// ValidateFile panics if internal assumptions have been broken. This
// should never happen. This method is intended for unit tests only.
//
// The assumptions here only hold true for files, not directories.
func (s *Clock) ValidateFile() {
	if !compareLE(s.mod, s.sync) {
		panic(fmt.Errorf("bad internal state: %s", s))
	}
}
