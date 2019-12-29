package strconvx_test

import (
	"testing"

	"bazil.org/bazil/cliutil/strconvx"
)

func TestInt(t *testing.T) {
	var x int
	err := strconvx.Parse(&x, "1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if g, e := x, int(1); g != e {
		t.Errorf("Unexpected result: %v != %v", g, e)
	}
}

func TestInt8(t *testing.T) {
	var x int8
	err := strconvx.Parse(&x, "1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if g, e := x, int8(1); g != e {
		t.Errorf("Unexpected result: %v != %v", g, e)
	}
}

func TestInt8OverFlow(t *testing.T) {
	var x int8
	err := strconvx.Parse(&x, "9000")
	if err == nil {
		t.Fatalf("Expected an error")
	}
	if g, e := err.Error(), `strconv.ParseInt: parsing "9000": value out of range`; g != e {
		t.Errorf("Wrong error message: %q != %q", g, e)
	}
}

var is32bit bool

func init() {
	var overflow uint = 1<<32 - 1
	overflow++
	is32bit = (overflow == 0)
}

func TestIntOverflow(t *testing.T) {
	if !is32bit {
		t.Skip("not on 32-bit architecture")
	}

	var x int
	err := strconvx.Parse(&x, "2147483648")
	if err == nil {
		t.Fatalf("Expected an error")
	}
	if g, e := err.Error(), `strconv.ParseInt: parsing "2147483648": value out of range`; g != e {
		t.Errorf("Wrong error message: %q != %q", g, e)
	}
}

func TestUintOverflow(t *testing.T) {
	if !is32bit {
		t.Skip("not on 32-bit architecture")
	}

	var x uint
	err := strconvx.Parse(&x, "4294967296")
	if err == nil {
		t.Fatalf("Expected an error")
	}
	if g, e := err.Error(), `strconv.ParseInt: parsing "4294967296": value out of range`; g != e {
		t.Errorf("Wrong error message: %q != %q", g, e)
	}
}
