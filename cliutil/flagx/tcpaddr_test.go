package flagx_test

import (
	"testing"

	"bazil.org/bazil/cliutil/flagx"
)

func TestTCPAddrEmpty(t *testing.T) {
	var a flagx.TCPAddr
	if err := a.Set(""); err != nil {
		t.Fatalf("empty TCPAddr.Set failed: %v", err)
	}
	if a.Addr != nil {
		t.Fatalf("empty TCPAddr is not nil: %v", a.Addr)
	}
}

func setTCPAddr(t testing.TB, value string) string {
	var a flagx.TCPAddr
	if err := a.Set(value); err != nil {
		t.Fatalf("TCPAddr.Set failed: %v", err)
	}
	return a.String()
}

func TestTCPAddrPort(t *testing.T) {
	if g, e := setTCPAddr(t, ":1234"), ":1234"; g != e {
		t.Errorf("unexpected TCPAddr: %q != %q", g, e)
	}
}

func TestTCPAddrHostPort(t *testing.T) {
	if g, e := setTCPAddr(t, "192.0.2.42:1234"), "192.0.2.42:1234"; g != e {
		t.Errorf("unexpected TCPAddr: %q != %q", g, e)
	}
}
