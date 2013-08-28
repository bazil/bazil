package subcommands

import (
	"flag"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"sync"

	"bazil.org/bazil/cliutil/positional"
)

type command struct {
	pkg string
	cmd interface{}
}

// Shell is a collection of commands, identified by the package that
// defines them.
type Shell struct {
	lock     sync.Mutex
	commands []command
}

func pkgName(cmd interface{}) string {
	val := reflect.ValueOf(cmd)
	pkg := val.Elem().Type().PkgPath()
	return pkg
}

// Register a new command.
//
// Each command is a singleton value of a unique type that has its
// address given to Shell.Register. Commands are identified by the
// package that defined the type; each command should be in a separate
// package.
//
// Commands may implement optional interfaces to enable more
// functionality. The recognized interfaces are:
//
//     - DescriptionGetter: short description to show when listing
//       subcommands
//     - FlagParser: will be called to process "-foo" style flags
//     - FlagSetter: used to recognize a flag.FlagSet and to control
//       its behavior
//     - VisiterAll: used to generate a help message for "-foo" style
//       flags
//
// Additionally, the command can define a struct named Arguments and
// bazil.org/bazil/cliutil/positional will be used to parse positional
// arguments into it.
func (s *Shell) Register(cmd interface{}) {
	pkg := pkgName(cmd)
	if pkg == "" {
		panic("Register called on unnamed type")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	s.commands = append(s.commands, command{pkg: pkg, cmd: cmd})
}

func (s *Shell) listSubcommands(pkg string) []command {
	s.lock.Lock()
	defer s.lock.Unlock()

	var found []command
	for _, c := range s.commands {
		if strings.HasPrefix(c.pkg, pkg+"/") {
			found = append(found, c)
		}
	}
	return found
}

// ErrMissingCommand indicates that a subcommand is needed but was not
// seen in the arguments.
type ErrMissingCommand struct{}

func (ErrMissingCommand) Error() string {
	return "missing mandatory subcommand"
}

// Parse examines the command line from args based on the top-level
// command cmd with the given name. It returns a Result that describes
// the result of the parsing, and an error. Result is valid even when
// error is not nil.
func (s *Shell) Parse(cmd interface{}, name string, args []string) (Result, error) {
	result := Result{
		parser: s,
	}
	pkg := pkgName(cmd)
	result.add(name, pkg, cmd)
	if pkg == "" {
		return result, fmt.Errorf("dispatch called for unnamed type: %v", cmd)
	}

dispatch:
	for {

		// parse flags
		parser, ok := cmd.(FlagParser)
		if !ok {
			parser = &flag.FlagSet{}
		}

		if fl, ok := parser.(FlagSetter); ok {
			// we provide our own usage text, so this FlagSet name is only
			// used in programmer error related messages
			fl.Init(pkg, flag.ContinueOnError)

			// flag has a bad habit of polluting stderr in .Parse() *and*
			// getting the formatting wrong (e.g. we want to include
			// command name as prefix), plus we want to control error &
			// usage output to unify all the possible error sources;
			// silence Parse and output the error and usage later, in the
			// caller
			fl.SetOutput(ioutil.Discard)
		}

		err := parser.Parse(args)
		if err != nil {
			return result, err
		}
		args = parser.Args()

		// see if we have positional args to parse
		if cmd != nil {
			argsField := reflect.ValueOf(cmd).Elem().FieldByName("Arguments")
			if argsField.IsValid() {
				argsI := argsField.Addr().Interface()
				err := positional.Parse(argsI, args)
				if err != nil {
					return result, err
				}
			}
		}

		var hasSub bool
		for _, subcmd := range s.listSubcommands(pkg) {
			if len(args) > 0 && subcmd.pkg == pkg+"/"+args[0] {
				// exact match
				pkg = subcmd.pkg
				cmd = subcmd.cmd
				result.add(args[0], pkg, cmd)
				args = args[1:]
				continue dispatch
			}

			if len(args) > 0 && strings.HasPrefix(subcmd.pkg, pkg+"/"+args[0]+"/") {
				// step in the right direction
				pkg = pkg + "/" + args[0]
				cmd = nil
				result.add(args[0], pkg, cmd)
				args = args[1:]
				continue dispatch
			}

			if strings.HasPrefix(subcmd.pkg, pkg+"/") {
				_, runnable := cmd.(Runner)
				if len(args) == 0 && !runnable {
					// we have subcommands but no args
					return result, ErrMissingCommand{}
				}

				hasSub = true
			}

		}

		if len(args) > 0 && hasSub {
			return result, fmt.Errorf("command not found: %v", args[0])
		}
		break
	}

	return result, nil
}
