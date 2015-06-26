package clock_test

import (
	"testing"

	"bazil.org/bazil/fs/clock"
)

// what would happen after paper figure 12, if the whole dir was
// synced A->B
func TestFigure12Continuation(t *testing.T) {
	a_dx := clock.Create(10, 1)
	a_dy := clock.Create(10, 1)
	a_d := &clock.Clock{}

	// new: no longer a partial sync
	a_d.UpdateSync(10, 1)
	a_d.UpdateFromChild(a_dx)
	a_d.UpdateFromChild(a_dy)
	a_dx.UpdateFromParent(a_d)
	a_dy.UpdateFromParent(a_d)
	if g, e := a_dx.String(), `{sync{} mod{10:1} create{10:1}}`; g != e {
		t.Errorf("bad state A d/x: %v != %v", g, e)
	}
	if g, e := a_dy.String(), `{sync{} mod{10:1} create{10:1}}`; g != e {
		t.Errorf("bad state A d/y: %v != %v", g, e)
	}
	if g, e := a_d.String(), `{sync{10:1} mod{10:1} create{}}`; g != e {
		t.Errorf("bad state A d: %v != %v", g, e)
	}

	b_dx := &clock.Clock{}
	b_dy := &clock.Clock{}
	b_d := &clock.Clock{}

	b_dx.ResolveTheirs(a_dx)
	b_d.UpdateFromChild(b_dx)
	b_dx.UpdateSync(11, 3)
	b_dy.UpdateSync(11, 3)
	// new: no longer a partial sync
	b_d.UpdateSync(11, 3)
	b_dx.UpdateFromParent(b_d)
	b_dy.UpdateFromParent(b_d)

	c_dx := &clock.Clock{}
	c_dy := &clock.Clock{}
	c_d := &clock.Clock{}

	c_dy.ResolveTheirs(a_dy)
	c_d.UpdateFromChild(c_dy)
	c_dx.UpdateSync(12, 3)
	c_dy.UpdateSync(12, 3)
	// new: no longer a partial sync
	c_d.UpdateSync(12, 3)
	c_dx.UpdateFromParent(c_d)
	c_dy.UpdateFromParent(c_d)

	// new things

	b_dy.ResolveTheirs(a_dy)
	b_d.ResolveOurs(a_d)
	b_d.UpdateFromChild(b_dy)
	b_dx.UpdateFromParent(b_d)
	b_dy.UpdateFromParent(b_d)

	c_dx.ResolveTheirs(a_dx)
	c_d.ResolveOurs(a_d)
	c_d.UpdateFromChild(c_dx)
	c_dx.UpdateFromParent(c_d)
	c_dy.UpdateFromParent(c_d)

	if g, e := b_dx.String(), `{sync{} mod{10:1} create{10:1}}`; g != e {
		t.Errorf("bad state B d/x: %v != %v", g, e)
	}
	if g, e := b_dy.String(), `{sync{} mod{10:1} create{10:1}}`; g != e {
		t.Errorf("bad state B d/y: %v != %v", g, e)
	}
	if g, e := b_d.String(), `{sync{10:1 11:3} mod{10:1} create{}}`; g != e {
		t.Errorf("bad state B d: %v != %v", g, e)
	}

	if g, e := c_dx.String(), `{sync{} mod{10:1} create{10:1}}`; g != e {
		t.Errorf("bad state C d/x: %v != %v", g, e)
	}
	if g, e := c_dy.String(), `{sync{} mod{10:1} create{10:1}}`; g != e {
		t.Errorf("bad state C d/y: %v != %v", g, e)
	}
	if g, e := c_d.String(), `{sync{10:1 12:3} mod{10:1} create{}}`; g != e {
		t.Errorf("bad state C d: %v != %v", g, e)
	}

	a_dx.ValidateFile(a_d)
	a_dy.ValidateFile(a_d)

	b_dx.ValidateFile(b_d)
	b_dy.ValidateFile(b_d)

	c_dx.ValidateFile(c_d)
	c_dy.ValidateFile(c_d)
}

func TestUpdateFromChildReturn(t *testing.T) {
	a_dx := clock.Create(10, 1)
	a_dy := clock.Create(10, 1)
	a_d := &clock.Clock{}

	a_d.UpdateSync(10, 1)
	if g, e := a_d.UpdateFromChild(a_dx), true; g != e {
		t.Errorf("UpdateFromChild return %v != %v", g, e)
	}
	if g, e := a_d.UpdateFromChild(a_dy), false; g != e {
		t.Errorf("UpdateFromChild return %v != %v", g, e)
	}
}

func TestRewritePeersSimple(t *testing.T) {
	c := clock.Create(10, 1)
	c.UpdateSync(11, 2)
	c.Update(12, 3)
	if g, e := c.String(), `{sync{10:1 11:2 12:3} mod{12:3} create{10:1}}`; g != e {
		t.Errorf("bad initial state: %v != %v", g, e)
	}

	m := map[clock.Peer]clock.Peer{
		10: 20,
		11: 12,
		12: 10,
	}
	if err := c.RewritePeers(m); err != nil {
		t.Fatalf("rewrite error: %v", err)
	}
	if g, e := c.String(), `{sync{20:1 12:2 10:3} mod{10:3} create{20:1}}`; g != e {
		t.Errorf("bad state: %v != %v", g, e)
	}
}

func TestRewritePeersBadPeer(t *testing.T) {
	c := clock.Create(10, 1)
	m := map[clock.Peer]clock.Peer{
		42: 13,
	}
	if g, e := c.RewritePeers(m), clock.ErrRewritePeerNotMapped; g != e {
		t.Errorf("wrong error: %v != %v", g, e)
	}
}
