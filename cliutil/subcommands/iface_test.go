package subcommands_test

import (
	"flag"
	"fmt"
	"os"

	"bazil.org/bazil/cliutil/subcommands"
)

func ExampleFlagParser() {
	type myCommand struct {
		flag.FlagSet
	}
}

func ExampleDescription() {
	type frobCommand struct {
		subcommands.Description
	}

	var frob = frobCommand{
		Description: "Frobnicate the bizbaz",
	}
	_ = frob
}

func ExampleSynopsis() {
	type frobCommand struct {
		subcommands.Synopsis
	}

	var frob = frobCommand{
		Synopsis: "POLARITY PARTICLE <FILE",
	}

	result, err := subcommands.Parse(&frob, "frob", []string{"reverse", "neutron"})
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	result.UsageTo(os.Stdout)
	// Output:
	// Usage:
	//   frob POLARITY PARTICLE <FILE
}

func ExampleSynopses() {
	type compressCommand struct {
		subcommands.Synopses
	}

	var compress = compressCommand{
		Synopses: []string{
			// compress refuses to output compressed data to a tty
			">FILE",
			"-o FILE",
		},
	}

	result, err := subcommands.Parse(&compress, "compress", []string{})
	if err != nil {
		fmt.Printf("error: %v", err)
		return
	}
	result.UsageTo(os.Stdout)
	// Output:
	// Usage:
	//   compress >FILE
	//   compress -o FILE
}
