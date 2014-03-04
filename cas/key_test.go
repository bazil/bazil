package cas_test

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"bazil.org/bazil/cas"
)

func TestKeyEmpty(t *testing.T) {
	buf := make([]byte, cas.KeySize)
	k := cas.NewKey(buf)
	if g, e := k, cas.Empty; g != e {
		t.Errorf("not Empty: %q != %q", g, e)
	}
	if g, e := k.String(), strings.Repeat("00", cas.KeySize); g != e {
		t.Errorf("bad key: %q != %q", g, e)
	}
	if g, e := k.IsSpecial(), true; g != e {
		t.Errorf("not Special: %v != %v", g, e)
	}
	if g, e := k.IsPrivate(), false; g != e {
		t.Errorf("bad Private: %v != %v", g, e)
	}
	if g, e := k.IsReserved(), false; g != e {
		t.Errorf("bad Reserved: %v != %v", g, e)
	}
}

func TestKeySimple(t *testing.T) {
	buf := bytes.Repeat([]byte("borketyBorkBORK!"), 4)
	k := cas.NewKey(buf)
	if g, e := k.String(), hex.EncodeToString(buf); g != e {
		t.Errorf("bad key: %q != %q", g, e)
	}
	if g, e := k.IsSpecial(), false; g != e {
		t.Errorf("bad Special: %v != %v", g, e)
	}
	if g, e := k.IsPrivate(), false; g != e {
		t.Errorf("bad Private: %v != %v", g, e)
	}
	if g, e := k.IsReserved(), false; g != e {
		t.Errorf("bad Reserved: %v != %v", g, e)
	}
}

func TestKeyBytes(t *testing.T) {
	buf := bytes.Repeat([]byte("borketyBorkBORK!"), 4)
	k := cas.NewKey(buf)
	if g, e := k.Bytes(), buf; !bytes.Equal(g, e) {
		t.Errorf("unexpected key data: %q %x", g, e)
	}
}

func TestKeyBadSize(t *testing.T) {
	buf := []byte("tooshort")
	defer func() {
		x := recover()
		switch i := x.(type) {
		case nil:
			t.Error("expected panic")
		case cas.BadKeySizeError:
			if g, e := i.Error(), "Key is bad length 8: 746f6f73686f7274"; g != e {
				t.Errorf("bad error message: %q != %q", g, e)
			}
		default:
			t.Errorf("expected BadKeySize: %v", x)
		}
	}()
	_ = cas.NewKey(buf)
}

func TestKeyInvalid(t *testing.T) {
	buf := make([]byte, cas.KeySize)
	buf[len(buf)-1] = 0x42
	k := cas.NewKey(buf)
	if g, e := k, cas.Invalid; g != e {
		t.Errorf("not Invalid: %q != %q", g, e)
	}
	if g, e := k.IsSpecial(), true; g != e {
		t.Errorf("not Special: %v != %v", g, e)
	}
	if g, e := k.IsPrivate(), false; g != e {
		t.Errorf("bad Private: %v != %v", g, e)
	}
	if g, e := k.IsReserved(), true; g != e {
		t.Errorf("not Reserved: %v != %v", g, e)
	}
}

func TestKeyInvalidPrivate(t *testing.T) {
	buf := make([]byte, cas.KeySize)
	buf[len(buf)-1] = 0x42
	k := cas.NewKeyPrivate(buf)
	if g, e := k, cas.Invalid; g != e {
		t.Errorf("not Invalid: %q != %q", g, e)
	}
}

func TestKeyNewPrivateNum(t *testing.T) {
	k := cas.NewKeyPrivateNum(31337)
	buf := k.Bytes()
	k2 := cas.NewKey(buf)
	if g, e := k2, cas.Invalid; g != e {
		t.Errorf("expected NewKey to give Invalid: %v", g)
	}
	k3 := cas.NewKeyPrivate(buf)
	if g, e := k3, k; g != e {
		t.Errorf("expected NewKeyPrivate to give original key: %v", g)
	}
	priv, ok := k3.Private()
	if !ok {
		t.Fatalf("expected Private to work: %v %v", priv, ok)
	}
	if g, e := priv, uint64(31337); g != e {
		t.Errorf("expected Private to match original: %v", g)
	}
	if g, e := k.IsSpecial(), true; g != e {
		t.Errorf("not Special: %v != %v", g, e)
	}
	if g, e := k.IsPrivate(), true; g != e {
		t.Errorf("not Private: %v != %v", g, e)
	}
	if g, e := k.IsReserved(), false; g != e {
		t.Errorf("bad Reserved: %v != %v", g, e)
	}
}

func TestKeyPrivateNotPriv(t *testing.T) {
	priv, ok := cas.Empty.Private()
	if ok {
		t.Fatalf("Empty should not be Private")
	}
	if g, e := priv, uint64(0); g != e {
		t.Errorf("expected zero value: %d != %d", g, e)
	}
}

func TestKeyUnmarshal(t *testing.T) {
	buf := bytes.Repeat([]byte("borketyBorkBORK!"), 4)
	k := cas.NewKey(buf)
	if g, e := k.Bytes(), buf; !bytes.Equal(g, e) {
		t.Errorf("unexpected key data: %q %x", g, e)
	}
}

func TestKeyUnmarshalBadShort(t *testing.T) {
	KEY := strings.Repeat("borketyBorkBORK!", 4)
	KEY = KEY[:len(KEY)-1]
	buf := []byte(KEY)
	var k cas.Key
	err := k.Unmarshal(buf)
	if err == nil {
		t.Fatalf("unmarshal should have failed: %v", k)
	}
	e, ok := err.(*cas.BadKeySizeError)
	if !ok {
		t.Fatalf("unmarshal error is of wrong type: %T: %v", err, err)
	}
	if g, w := string(e.Key), KEY; g != w {
		t.Errorf("BadKeySizeError Key is wrong: %x != %x", g, w)
	}
}

func TestKeyUnmarshalBadLong(t *testing.T) {
	KEY := strings.Repeat("borketyBorkBORK!", 4) + "x"
	buf := []byte(KEY)
	var k cas.Key
	err := k.Unmarshal(buf)
	if err == nil {
		t.Fatalf("unmarshal should have failed: %v", k)
	}
	e, ok := err.(*cas.BadKeySizeError)
	if !ok {
		t.Fatalf("unmarshal error is of wrong type: %T: %v", err, err)
	}
	if g, w := string(e.Key), KEY; g != w {
		t.Errorf("BadKeySizeError Key is wrong: %x != %x", g, w)
	}
}

func TestKeyMarshalTo(t *testing.T) {
	buf := bytes.Repeat([]byte("borketyBorkBORK!"), 4)
	k := cas.NewKey(buf)
	out := make([]byte, 100)
	n, err := k.MarshalTo(out)
	if err != nil {
		t.Fatal(err)
	}
	if g, e := n, cas.KeySize; g != e {
		t.Errorf("unexpected MarshalTo size: %q %x", g, e)
	}
	if g, e := string(out[:n]), string(buf); g != e {
		t.Errorf("unexpected MarshalTo data: %q != %q", g, e)
	}
}
