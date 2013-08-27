package positional_test

import (
	"fmt"
	"testing"

	"bazil.org/bazil/cliutil/positional"
)

func Example() {
	type Inventory struct {
		Item string
		positional.Optional
		Count int
	}

	porch := Inventory{}
	err := positional.Parse(&porch, []string{"cat", "3"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("You have %d %s(s) on the porch\n", porch.Count, porch.Item)

	house := Inventory{
		Count: 1,
	}
	err = positional.Parse(&house, []string{"dog"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Printf("You have %d %s(s) in the house\n", house.Count, house.Item)

	// Output:
	// You have 3 cat(s) on the porch
	// You have 1 dog(s) in the house
}

func TestParseEmpty(t *testing.T) {
	var args struct {
	}
	err := positional.Parse(&args, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseTooMany(t *testing.T) {
	var args struct {
		Foo string
	}
	err := positional.Parse(&args, []string{"one", "two"})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if _, ok := err.(positional.ErrTooManyArgs); !ok {
		t.Errorf("unexpected error type: %T", err)
	}
	if err.Error() != "too many arguments" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	if g, e := args.Foo, "one"; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
}

func TestParseMandatory(t *testing.T) {
	var args struct {
		Foo string
	}
	err := positional.Parse(&args, []string{"one"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := args.Foo, "one"; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
}

func TestParseMandatoryMissing(t *testing.T) {
	var args struct {
		Foo string
	}
	err := positional.Parse(&args, []string{})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if _, ok := err.(positional.ErrMissingMandatoryArg); !ok {
		t.Errorf("unexpected error type: %T", err)
	}
	if err.Error() != "missing mandatory argument: FOO" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	if g, e := args.Foo, ""; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
}

func TestParseOptional(t *testing.T) {
	var args struct {
		positional.Optional
		Foo string
	}
	err := positional.Parse(&args, []string{"one"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := args.Foo, "one"; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
}

func TestParseOptionalMissing(t *testing.T) {
	var args struct {
		positional.Optional
		Foo string
	}
	err := positional.Parse(&args, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := args.Foo, ""; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
}

func TestParseBoth(t *testing.T) {
	var args struct {
		Foo string
		positional.Optional
		Bar string
	}
	err := positional.Parse(&args, []string{"one", "two"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := args.Foo, "one"; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
	if g, e := args.Bar, "two"; g != e {
		t.Errorf("unexpected value for Bar: %q != %q", g, e)
	}
}

func TestParseBothMissingOptional(t *testing.T) {
	var args struct {
		Foo string
		positional.Optional
		Bar string
	}
	err := positional.Parse(&args, []string{"one"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := args.Foo, "one"; g != e {
		t.Errorf("unexpected value for Foo: %q != %q", g, e)
	}
	if g, e := args.Bar, ""; g != e {
		t.Errorf("unexpected value for Bar: %q != %q", g, e)
	}
}

func TestParseMandatoryPlural(t *testing.T) {
	var args struct {
		Foo []string
	}
	err := positional.Parse(&args, []string{"one", "two"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := len(args.Foo), 2; g != e {
		t.Errorf("unexpected length for Foo: %d != %d", g, e)
	}
	if g, e := args.Foo[0], "one"; g != e {
		t.Errorf("unexpected value for Foo[0]: %q != %q", g, e)
	}
	if g, e := args.Foo[1], "two"; g != e {
		t.Errorf("unexpected value for Foo[1]: %q != %q", g, e)
	}
}

func TestParseInt(t *testing.T) {
	var args struct {
		Foo int
	}
	err := positional.Parse(&args, []string{"1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := args.Foo, 1; g != e {
		t.Errorf("unexpected value for Foo: %d != %d", g, e)
	}
}

func TestParsePtrInt(t *testing.T) {
	var args struct {
		Foo *int
	}
	err := positional.Parse(&args, []string{"1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if args.Foo == nil {
		t.Fatal("unexpeced nil value for Foo")
	}
	if g, e := *args.Foo, 1; g != e {
		t.Errorf("unexpected value for Foo: %d != %d", g, e)
	}
}
