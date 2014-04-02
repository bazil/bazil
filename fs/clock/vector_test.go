package clock

import "testing"

func TestStringEmpty(t *testing.T) {
	var v vector
	if g, e := v.String(), `{}`; g != e {
		t.Errorf("bad stringer: %#v -> %s != %s", v, g, e)
	}
}

func TestStringSimple(t *testing.T) {
	var v vector
	v.update(10, 1)
	v.update(11, 1)
	v.update(10, 2)
	if g, e := v.String(), `{10:2 11:1}`; g != e {
		t.Errorf("bad stringer: %#v -> %s != %s", v, g, e)
	}
}

func TestMergeSimple(t *testing.T) {
	var a vector
	a.update(10, 1)
	a.update(11, 1)
	a.update(10, 2)
	var b vector
	b.merge(a)
	// trigger bugs if they accidentally share state
	a.update(10, 3)

	if g, e := b.String(), `{10:2 11:1}`; g != e {
		t.Errorf("bad merge: %s != %s", g, e)
	}
}

func TestRebaseSimple(t *testing.T) {
	var a vector
	a.update(10, 1)
	a.update(11, 1)
	var b vector
	b.merge(a)
	a.update(10, 2)
	b.update(12, 3)

	a.rebase(b)

	if g, e := b.String(), `{10:1 11:1 12:3}`; g != e {
		t.Errorf("bad rebase: %s != %s", g, e)
	}
	if g, e := a.String(), `{10:2}`; g != e {
		t.Errorf("bad rebase: %s != %s", g, e)
	}
}

func TestCompareLEEmpty(t *testing.T) {
	var a vector
	var b vector
	if g, e := compareLE(a, b), true; g != e {
		t.Errorf("bad comparison: %s is to %s -> %v != %v", a, b, g, e)
	}
}

func TestCompareLESimple(t *testing.T) {
	var a vector
	a.update(10, 1)
	var b vector
	b.merge(a)
	if g, e := compareLE(a, b), true; g != e {
		t.Errorf("bad comparison: %s is to %s -> %v != %v", a, b, g, e)
	}
}

func TestCompareLEConcurrent(t *testing.T) {
	var a vector
	a.update(10, 1)
	var b vector
	b.update(11, 1)
	if g, e := compareLE(a, b), false; g != e {
		t.Errorf("bad comparison: %s is to %s -> %v != %v", a, b, g, e)
	}
}
