package idpool

// Pool keeps track of uint64 identifiers.
//
// Caller is reponsible for locking.
type Pool struct {
	// x<next are reserved, unless free says otherwise; next==0 means
	// none are reserved.
	next uint64

	free []uint64
}

// Get returns an available id from the pool.
//
// Get panics if the pool is exhausted; 2**64 should be enough for
// everyone.
func (p *Pool) Get() uint64 {
	var id uint64
	if n := len(p.free); n > 0 {
		id = p.free[n-1]
		p.free = p.free[:n-1]
		return id
	}

	r := p.next
	p.next += 1
	// TODO panic somewhere around here if wraparound!!
	return r
}

// Put returns an id to the pool.
func (p *Pool) Put(id uint64) {
	p.free = append(p.free, id)
}

// SetMinimum sets the minimum value that will be returned. This is
// useful for reserving some ids for special use.
func (p *Pool) SetMinimum(min uint64) {
	if p.next < min {
		p.next = min
	}

	// make sure the reserved ids aren't returned from the free list
	// either
	for i, id := range p.free {
		if id < min {
			n := len(p.free) - 1
			if i < n {
				// move last item here so we can just truncate
				p.free[i] = p.free[n]
			}
			p.free = p.free[:n]
		}
	}
}
