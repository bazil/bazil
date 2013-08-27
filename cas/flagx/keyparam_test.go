package flagx_test

import (
	"encoding/hex"
	"flag"
	"fmt"
	"testing"

	"bazil.org/bazil/cas"
	"bazil.org/bazil/cas/flagx"
)

func ExampleKeyParam() {
	var kp flagx.KeyParam
	var fl flag.FlagSet

	fl.Var(&kp, "key", "blah blah")
	fl.Parse([]string{"-key=095920158295e252b3cb4728713a987875d210eca00d5b69e89e1d2e2e473153ec3de87ada31787b0d5080cdb0f7dcf15ea1f03cec5fef76df027bcc7d57b337"})
	k := kp.Key()
	fmt.Println(k)

	// Output:
	// 095920158295e252b3cb4728713a987875d210eca00d5b69e89e1d2e2e473153ec3de87ada31787b0d5080cdb0f7dcf15ea1f03cec5fef76df027bcc7d57b337
}

func TestKeyParamFlagSetOk(t *testing.T) {
	var kp flagx.KeyParam
	v := flag.Value(&kp)
	const hexKey = "095920158295e252b3cb4728713a987875d210eca00d5b69e89e1d2e2e473153ec3de87ada31787b0d5080cdb0f7dcf15ea1f03cec5fef76df027bcc7d57b337"
	err := v.Set(hexKey)
	if err != nil {
		t.Fatalf("unexpected error from Set: %v", err)
	}
	if g, e := v.String(), hexKey; g != e {
		t.Errorf("bad string value: %v", g)
	}
}

func TestKeyParamFlagSetNotHex(t *testing.T) {
	var kp flagx.KeyParam
	v := flag.Value(&kp)
	const hexKey = "i am not even hex!"
	err := v.Set(hexKey)
	if err == nil {
		t.Fatalf("expected an error from Set: %v", err)
	}
	if _, ok := err.(hex.InvalidByteError); !ok {
		t.Fatalf("bad error type Set: %T: %v", err, err)
	}
	if g, e := err.Error(), "encoding/hex: invalid byte: U+0069 'i'"; g != e {
		t.Errorf("bad string value: %v", g)
	}
}

func TestKeyParamFlagSetTooShort(t *testing.T) {
	var kp flagx.KeyParam
	v := flag.Value(&kp)
	const hexKey = "ff"
	err := v.Set(hexKey)
	if err == nil {
		t.Fatalf("expected an error from Set: %v", err)
	}
	err2, ok := err.(*cas.BadKeySizeError)
	if !ok {
		t.Fatalf("bad error type Set: %T: %v", err, err)
	}
	if g, e := string(err2.Key), "\xFF"; g != e {
		t.Errorf("bad error detail: %x", g)
	}
	if g, e := err2.Error(), "Key is bad length 1: ff"; g != e {
		t.Errorf("bad string value: %v", g)
	}
}

func TestKeyParamFlagSetInvalid(t *testing.T) {
	var kp flagx.KeyParam
	v := flag.Value(&kp)
	var hexKey = cas.Invalid.String()
	err := v.Set(hexKey)
	if err == nil {
		t.Fatalf("expected an error from Set: %v", err)
	}
	if g, e := err.Error(), "bad key format"; g != e {
		t.Errorf("bad string value: %v", g)
	}
}
