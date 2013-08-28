// Package subcommands is a framework for creating command-based
// interfaces with hierarchical commands.
//
// Each command is in a separate package, and the package import path
// hierarchy is used to create the command hierarchy. For example, the
// package github.com/myuser/myproject/foo/bar/baz could be the
// subcommand "bar baz" under the top-level command "foo".
//
// TODO(tv) this API is not considered final yet
//
// BUG(tv) multiple use leaves state around -> not currently useful
// for more than command line parsing. maybe should instantiate new
// command values, instead of using the registered ones?
package subcommands
