package idpool_test

import (
	"testing"

	"bazil.org/bazil/idpool"
)

func TestSimple(t *testing.T) {
	p := idpool.Pool{}
	if g, e := p.Get(), uint64(0); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	if g, e := p.Get(), uint64(1); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	if g, e := p.Get(), uint64(2); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	p.Put(1)
	if g, e := p.Get(), uint64(1); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
}

func TestMinimum(t *testing.T) {
	p := idpool.Pool{}
	p.SetMinimum(3)
	if g, e := p.Get(), uint64(3); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
}

func TestMinimumWithFree(t *testing.T) {
	p := idpool.Pool{}
	if g, e := p.Get(), uint64(0); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	if g, e := p.Get(), uint64(1); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	if g, e := p.Get(), uint64(2); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	if g, e := p.Get(), uint64(3); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
	p.Put(1)
	p.Put(0)
	p.SetMinimum(2)
	if g, e := p.Get(), uint64(4); g != e {
		t.Errorf("Bad Get result: %d != %d", g, e)
	}
}
