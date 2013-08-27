package positional_test

import (
	"fmt"
	"testing"

	"bazil.org/bazil/cliutil/positional"
)

func ExampleUsage() {
	var args struct {
		Op string `positional:",metavar=ACTION"`
		positional.Optional
		Path []string
	}
	usage := positional.Usage(&args)
	fmt.Printf("Usage: doit [-v] %s\n", usage)
	// Output:
	// Usage: doit [-v] ACTION [PATH..]
}

func TestUsageEmpty(t *testing.T) {
	var args struct {
	}
	usage := positional.Usage(&args)
	if g, e := usage, ""; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}

func TestUsageMandatory(t *testing.T) {
	var args struct {
		Foo string
	}
	usage := positional.Usage(&args)
	if g, e := usage, "FOO"; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}

func TestUsageMandatoryTwo(t *testing.T) {
	var args struct {
		Foo string
		Bar string
	}
	usage := positional.Usage(&args)
	if g, e := usage, "FOO BAR"; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}

func TestUsageOptional(t *testing.T) {
	var args struct {
		positional.Optional
		Foo string
	}
	usage := positional.Usage(&args)
	if g, e := usage, "[FOO]"; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}

func TestUsageOptionalTwo(t *testing.T) {
	var args struct {
		positional.Optional
		Foo string
		Bar string
	}
	usage := positional.Usage(&args)
	if g, e := usage, "[FOO [BAR]]"; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}

func TestUsageMandatoryPlural(t *testing.T) {
	var args struct {
		Foo []string
	}
	usage := positional.Usage(&args)
	if g, e := usage, "FOO.."; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}

func TestUsageOptionalPlural(t *testing.T) {
	var args struct {
		positional.Optional
		Foo []string
	}
	usage := positional.Usage(&args)
	if g, e := usage, "[FOO..]"; g != e {
		t.Errorf("unexpected usage: %q != %q", g, e)
	}
}
