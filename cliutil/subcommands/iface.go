package subcommands

import (
	"flag"
	"io"
)

// FlagParser is implemented by commands that wish to process their
// arguments before subcommand traversal continues.
//
// The typical way to implement this is to embed flag.FlagSet in the
// command struct.
type FlagParser interface {
	Parse(args []string) error
	Args() []string
}

// FlagSetter is used to recognize a flag.FlagSet (even when embedded
// in a struct). It is used to disable the undesired behavior of the
// flag library: to prevent program termination and control stderr
// output.
type FlagSetter interface {
	Init(name string, errorHandling flag.ErrorHandling)
	SetOutput(w io.Writer)
}

// VisiterAll is an interface that lets commands report what "-foo"
// style flags they support. This is used for help output.
//
// The typical way to implement this is to embed flag.FlagSet in the
// command struct.
type VisiterAll interface {
	VisitAll(fn func(*flag.Flag))
}

// Runner is used as a marker interface to distinguish commands that
// are valid by themselves, even when they have subcommands. Run is
// never actually called.
//
// It is also intended as a convenience interface for the caller, to
// convert the interface{} returned from Result.Command() into
// something that can be acted upon.
type Runner interface {
	Run() error
}

// DescriptionGetter is used to give a short description of the
// command when showing a list of subcommands.
//
// The typical way to implement this is to embed Description in the
// command struct, and give the description when declaring the
// variable. See Description for an example.
type DescriptionGetter interface {
	GetDescription() string
}

// SynopsesGetter is used to give a list of synopses snippets, short
// summaries of the arguments that can be passed in.
//
// The typical way to implement this is to embed Synopsis or Synopses
// in the command struct, and give the synopses when declaring the
// variable. See Synopsis and Synopses for examples.
type SynopsesGetter interface {
	GetSynopses() []string
}

// Overviewer is used to give an overview explanation of the command.
// This is displayed in the usage message after the synopsis, before
// options.
//
// The typical way to implement this is to embed Overview in the
// command struct, and give the overview when declaring the variable.
// See Overview examples.
//
// Leading newlines are removed, to make writing multi-line raw
// strings more convenient. A trailing newline is added if one is not
// present, to make writing single-line strings more convenient.
type Overviewer interface {
	GetOverview() string
}
