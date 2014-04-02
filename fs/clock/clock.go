// Package clock implements a logical clock, tracking changes at multiple peers.
//
// Is is inspired by the paper "File Synchronization with Vector Time
// Pairs" and the Tra software. This structure avoids the need for
// tombstones.
//
//  http://publications.csail.mit.edu/tmp/MIT-CSAIL-TR-2005-014.pdf
//  http://swtch.com/tra/
package clock

import "fmt"

// A Peer is a replica that may create new versions of the tracked
// data. Peers are identified by small unsigned integers, for
// efficiency.
type Peer uint32

// Epoch is a logical clock timestamp. Time 0 is never valid.
type Epoch uint64

// Clock is a logical clock.
//
// The zero value is a valid empty clock.
type Clock struct {
	sync vector
	mod  vector
}

// Update adds or updates the version vector entry for id to point to
// time now.
//
// Caller guarantees if an entry exists for id already, now is greater
// than or equal to the old value.
func (s *Clock) Update(id Peer, now Epoch) {
	s.mod.update(id, now)
	s.sync.update(id, now)
}

// ResolveTheirs records a conflict resolution in favor of other.
func (s *Clock) ResolveTheirs(other *Clock) {
	s.mod = vector{}
	s.mod.merge(other.mod)
	s.sync.merge(other.mod)
}

// ResolveOurs records a conflict resolution in favor of us.
func (s *Clock) ResolveOurs(other *Clock) {
	// no change to s.mod
	s.sync.merge(other.mod)
}

// ResolveNew records a conflict resolution in favor of newly created
// content.
func (s *Clock) ResolveNew(other *Clock) {
	s.mod.merge(other.mod)
	s.sync.merge(other.mod)
}

func (s Clock) String() string {
	return fmt.Sprintf("{sync%s mod%s}", s.sync, s.mod)
}

// Action is a suggested action to take to combine two data items.
//
// The zero value of Action is not valid.
type Action int

//go:generate stringer -type=Action

const (
	InvalidAction Action = iota
	// Copy means that the incoming version is newer, and its data
	// should be used.
	Copy
	// Nothing means the local version is newer (or same), and data
	// should not change.
	Nothing
	// Conflict means the two versions have diverged.
	Conflict
)

// Sync returns what action receiving state from A to B should cause
// us to take.
func Sync(a, b *Clock) Action {
	if compareLE(a.mod, b.sync) {
		return Nothing
	}

	if compareLE(b.mod, a.sync) {
		return Copy
	}

	return Conflict
}
