package subcommands_test

import (
	"flag"
	"testing"

	"bazil.org/bazil/cliutil/positional"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/cliutil/subcommands/test/calc"
	"bazil.org/bazil/cliutil/subcommands/test/calc/sum"
)

func TestParseEmpty(t *testing.T) {
	_, err := subcommands.Parse(&calc.Calc, "calc", []string{})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if _, ok := err.(subcommands.ErrMissingCommand); !ok {
		t.Errorf("unexpected error type: %T", err)
	}
	if err.Error() != "missing mandatory subcommand" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}

func TestParseMissingArgs(t *testing.T) {
	result, err := subcommands.Parse(&calc.Calc, "calc", []string{"sum", "1"})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if _, ok := err.(positional.ErrMissingMandatoryArg); !ok {
		t.Errorf("unexpected error type: %T", err)
	}
	if err.Error() != "missing mandatory argument: B" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	commands := result.ListCommands()
	if g, e := commands[len(commands)-1], &sum.Sum; g != e {
		t.Fatalf("unexpected dispatch: %#v != %#v", g, e)
	}
	if g, e := sum.Sum.Arguments.A, 1; g != e {
		t.Errorf("unexpected arg A: %#v != %#v", g, e)
	}
	// did not get set
	if g, e := sum.Sum.Arguments.B, 0; g != e {
		t.Errorf("unexpected arg B: %#v != %#v", g, e)
	}
}

func TestParseSimple(t *testing.T) {
	result, err := subcommands.Parse(&calc.Calc, "calc", []string{"sum", "1", "3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if g, e := result.Name(), "calc sum"; g != e {
		t.Fatalf("unexpected command name: %q != %q", g, e)
	}
	commands := result.ListCommands()
	if g, e := commands[len(commands)-1], &sum.Sum; g != e {
		t.Fatalf("unexpected dispatch: %#v != %#v", g, e)
	}
	if g, e := sum.Sum.Arguments.A, 1; g != e {
		t.Errorf("unexpected arg A: %#v != %#v", g, e)
	}
	if g, e := sum.Sum.Arguments.B, 3; g != e {
		t.Errorf("unexpected arg B: %#v != %#v", g, e)
	}
}

func TestParseFlags(t *testing.T) {
	result, err := subcommands.Parse(&calc.Calc, "calc", []string{"sum", "-frobnicate", "1", "2"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	commands := result.ListCommands()
	if g, e := commands[len(commands)-1], &sum.Sum; g != e {
		t.Fatalf("Unexpected dispatch: %#v != %#v", g, e)
	}
	if g, e := sum.Sum.Config.Frob, true; g != e {
		t.Errorf("Unexpected flag value: %#v != %#v", g, e)
	}
}

func TestParseHelp(t *testing.T) {
	result, err := subcommands.Parse(&calc.Calc, "calc", []string{"-help"})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if err != flag.ErrHelp {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	commands := result.ListCommands()
	if g, e := commands[len(commands)-1], &calc.Calc; g != e {
		t.Fatalf("unexpected dispatch: %#v != %#v", g, e)
	}
}

func TestParseHelpSub(t *testing.T) {
	result, err := subcommands.Parse(&calc.Calc, "calc", []string{"sum", "-help"})
	if err == nil {
		t.Fatalf("expected an error")
	}
	if err != flag.ErrHelp {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	commands := result.ListCommands()
	if g, e := commands[len(commands)-1], &sum.Sum; g != e {
		t.Fatalf("unexpected dispatch: %#v != %#v", g, e)
	}
}
