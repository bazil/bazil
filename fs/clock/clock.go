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
// The zero value is a valid empty clock, but most callers should call
// Create to get the creation time set up.
type Clock struct {
	sync vector
	mod  vector
	// create is always of length 1
	//
	// TODO could make it an item not a vector, but then we can't
	// use compareLE.
	create vector
}

// Create returns a new Vector Pair that knows it was created by id at
// time now.
func Create(id Peer, now Epoch) *Clock {
	c := &Clock{
		sync:   vector{list: []item{{id: id, t: now}}},
		mod:    vector{list: []item{{id: id, t: now}}},
		create: vector{list: []item{{id: id, t: now}}},
	}
	return c
}

// Update adds or updates the version vector entry for id to point to
// time now.
//
// As an optimization, it removes all the other modification time
// entries. This is only safe for files, not directories; see section
// 3.5.2 "Encoding Modification Times" of the Tra paper.
//
// Caller guarantees that one of the following is true:
//     - now is greater than any old value seen for peer id
//     - now is equal to an earlier value for this peer id, and no
//       other peer ids have been updated since that Update
func (s *Clock) Update(id Peer, now Epoch) {
	s.mod.updateSimplify(id, now)
	s.sync.update(id, now)
}

// UpdateParent is like Update, but does not simplify the modification
// time version vector. It is safe to use for directories and other
// entities where updates are not necessarily sequenced.
//
// Caller guarantees if an entry exists for id already, now is greater
// than or equal to the old value.
func (s *Clock) UpdateParent(id Peer, now Epoch) {
	s.mod.update(id, now)
	s.sync.update(id, now)
}

// UpdateSync updates the sync time only for id to point to time now.
//
// Caller guarantees if an entry exists for id already, now is greater
// than or equal to the old value.
func (s *Clock) UpdateSync(id Peer, now Epoch) {
	s.sync.update(id, now)
}

// UpdateFromChild tracks child modification times in the parent.
func (s *Clock) UpdateFromChild(child *Clock) {
	s.mod.merge(child.mod)
}

// UpdateFromParent simplifies child sync times based on the parent.
func (s *Clock) UpdateFromParent(parent *Clock) {
	s.sync.rebase(parent.sync)
}

// Tombstone changes clock into a tombstone.
func (s *Clock) Tombstone() {
	// vpair paper section 3.3.2
	s.mod = vector{}
	s.create = vector{}
}

// ResolveTheirs records a conflict resolution in favor of other.
func (s *Clock) ResolveTheirs(other *Clock) {
	s.mod = vector{}
	s.mod.merge(other.mod)
	s.sync.merge(other.sync)
	// vpair paper is silent on what to do with create times; if we
	// don't do this, s.create can remain empty
	if len(s.create.list) == 0 {
		s.create.merge(other.create)
	}
}

// ResolveOurs records a conflict resolution in favor of us.
func (s *Clock) ResolveOurs(other *Clock) {
	// no change to s.mod
	s.sync.merge(other.sync)
	// no change to s.create
}

// ResolveNew records a conflict resolution in favor of newly created
// content.
func (s *Clock) ResolveNew(other *Clock) {
	s.mod.merge(other.mod)
	s.sync.merge(other.sync)
	// no change to s.create
}

func (s Clock) String() string {
	return fmt.Sprintf("{sync%s mod%s create%s}", s.sync, s.mod, s.create)
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

// SyncToMissing returns what action receiving state from A to B
// should cause us to take. B does not exist currently.
func SyncToMissing(a, b *Clock) Action {
	if compareLE(a.mod, b.sync) {
		return Nothing
	}

	if !compareLE(a.create, b.sync) {
		return Copy
	}

	return Conflict
}
