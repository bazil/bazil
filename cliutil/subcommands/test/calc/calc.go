package calc

import (
	"flag"

	"bazil.org/bazil/cliutil/subcommands"
)

type calc struct {
	flag.FlagSet
	Config struct {
		Verbose bool
	}
}

// Calc is exported so the unit tests can inspect it.
var Calc calc

var _ = subcommands.FlagParser(&Calc)

func init() {
	Calc.BoolVar(&Calc.Config.Verbose, "v", false, "verbose output")
	subcommands.Register(&Calc)
}
