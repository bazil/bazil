package clock

import (
	"sort"
	"strconv"
)

type item struct {
	id Peer
	t  Epoch
}

type vector struct {
	list []item
}

func (v vector) Len() int {
	return len(v.list)
}

func (v vector) Less(i, j int) bool {
	return v.list[i].id < v.list[j].id
}

func (v vector) Swap(i, j int) {
	v.list[i], v.list[j] = v.list[j], v.list[i]
}

var _ sort.Interface = (*vector)(nil)

func (v vector) String() string {
	buf := []byte{'{'}
	for i, x := range v.list {
		if i > 0 {
			buf = append(buf, ' ')
		}
		buf = strconv.AppendUint(buf, uint64(x.id), 10)
		buf = append(buf, ':')
		buf = strconv.AppendUint(buf, uint64(x.t), 10)
	}
	buf = append(buf, '}')
	return string(buf)
}

// ensure id is in the list, adding it there if necessary
func (v *vector) add(id Peer) int {
	i := sort.Search(len(v.list), func(i int) bool {
		return v.list[i].id >= id
	})
	if i < len(v.list) && v.list[i].id == id {
		// found
		return i
	}

	// not found; insert at i

	// let append worry about reallocation; dummy placeholder
	v.list = append(v.list, item{})

	// shuffle to insert
	copy(v.list[i+1:], v.list[i:])
	v.list[i] = item{id: id}
	return i
}

func (v *vector) update(id Peer, now Epoch) {
	i := v.add(id)
	v.list[i].t = now
}

// updateSimplify adds id at time now to the vector, knowing that only
// one entry needs to be preserved.
func (v *vector) updateSimplify(id Peer, now Epoch) {
	v.list = []item{{id: id, t: now}}
}

func (v *vector) merge(other vector) bool {
	changed := false
	for _, o := range other.list {
		i := v.add(o.id)
		if v.list[i].t < o.t {
			changed = true
			v.list[i].t = o.t
		}
	}
	return changed
}

// rebase sets v to only include the items not in other.
func (v *vector) rebase(other vector) {
	myIdx := 0
	otherIdx := 0
outer:
	for myIdx < len(v.list) && otherIdx < len(other.list) {
		switch {
		case v.list[myIdx].id < other.list[otherIdx].id:
			// we have an item other does not, keep it
			myIdx++
			continue outer
		case v.list[myIdx].id > other.list[otherIdx].id:
			// other has extra items, never mind
			otherIdx++
			continue outer
		default:
			// same ids
			if v.list[myIdx].t > other.list[otherIdx].t {
				// need to preserve entry
				myIdx++
			} else {
				// can simplify, remove the entry
				copy(v.list[myIdx:], v.list[myIdx+1:])
				v.list = v.list[:len(v.list)-1]
			}
			otherIdx++
			continue outer
		}
	}
}

func (v *vector) rewritePeers(m map[Peer]Peer) error {
	for i := range v.list {
		id, ok := m[v.list[i].id]
		if !ok {
			return ErrRewritePeerNotMapped
		}
		v.list[i].id = id
	}
	sort.Sort(v)
	return nil
}

// compareLE tests if A <= B.
func compareLE(a, b vector) bool {
	aIdx := 0
	bIdx := 0
outer:
	for aIdx < len(a.list) && bIdx < len(b.list) {
		switch {
		case a.list[aIdx].id < b.list[bIdx].id:
			// a has an id b does not -> B<A or A||B
			return false
		case a.list[aIdx].id > b.list[bIdx].id:
			// b has an id a does not -> A<B or A||B, keep going
			bIdx++
			continue outer
		default:
			// same ids
			if a.list[aIdx].t > b.list[bIdx].t {
				// B<A or A||B
				return false
			}
			aIdx++
			bIdx++
		}
	}

	if len(a.list) > aIdx {
		// a has an id b does not -> B<A or A||B
		return false
	}

	// there may be elements left in B, but we don't care whether A==B
	// or A<B

	return true
}
