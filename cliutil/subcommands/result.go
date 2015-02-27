package subcommands

import (
	"flag"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"

	"bazil.org/bazil/cliutil/positional"
)

type atom struct {
	pkg string
	cmd interface{}
}

// Result is the result of parsing the arguments given to a command.
//
// TODO(tv) this API may change a lot
type Result struct {
	parser *Shell
	name   string
	list   []atom
}

func (r *Result) add(name string, pkg string, cmd interface{}) {
	if r.name != "" {
		name = " " + name
	}
	r.name += name
	r.list = append(r.list, atom{
		pkg: pkg,
		cmd: cmd,
	})
}

// Name returns the full name of the subcommand being executed,
// including all the parents.
func (r *Result) Name() string {
	return r.name
}

// ListCommands returns a list of the subcommands encountered. Index 0
// is the topmost parent, last item is the active subcommand.
func (r *Result) ListCommands() []interface{} {
	l := make([]interface{}, len(r.list))
	for i, a := range r.list {
		l[i] = a.cmd
	}
	return l
}

func (r *Result) last() *atom {
	// guaranteed len >0 because top level is always added here
	return &r.list[len(r.list)-1]
}

// Usage writes a usage message for the active subcommand to standard
// error.
func (r *Result) Usage() {
	r.UsageTo(os.Stderr)
}

// UsageTo writes a usage message for the active subcommand to the
// given Writer.
func (r *Result) UsageTo(w io.Writer) {
	fmt.Fprintf(w, "Usage:\n")
	cmd := r.last().cmd

	var synopses []string
	if s, ok := cmd.(SynopsesGetter); ok {
		synopses = s.GetSynopses()
	} else {
		syn := []string{}

		if v, ok := cmd.(VisiterAll); ok {
			var opts bool
			v.VisitAll(func(flag *flag.Flag) { opts = true })
			if opts {
				syn = append(syn, "[OPT..]")
			}
		}

		var didArgs bool
		if cmd != nil {
			argsField := reflect.ValueOf(cmd).Elem().FieldByName("Arguments")
			if argsField.IsValid() {
				argsI := argsField.Addr().Interface()
				syn = append(syn, positional.Usage(argsI))
				didArgs = true
			}
		}

		if !didArgs {
			var hasSub bool
			pkg := r.last().pkg
			for _, subcmd := range r.parser.listSubcommands(pkg) {
				if strings.HasPrefix(subcmd.pkg, pkg+"/") {
					hasSub = true
				}
			}
			if hasSub {
				syn = append(syn, "COMMAND..")
			}
		}

		synopses = []string{
			strings.Join(syn, " "),
		}
	}
	for _, s := range synopses {
		fmt.Fprintf(w, "  %s %s\n", r.name, s)
	}

	if o, ok := cmd.(Overviewer); ok {
		s := o.GetOverview()
		s = strings.Trim(s, "\n")
		fmt.Fprintf(w, "\n%s\n", s)
	}

	if v, ok := cmd.(VisiterAll); ok {
		var header bool
		v.VisitAll(func(flag *flag.Flag) {
			if !header {
				fmt.Fprintf(w, "\nOptions:\n")
				header = true
			}

			fmt.Fprintf(w, "  -%s=%s: %s\n", flag.Name, flag.DefValue, flag.Usage)
		})
	}

	subs := r.parser.listSubcommands(r.last().pkg)
	if len(subs) > 0 {
		fmt.Fprintf(w, "\nCommands:\n")

		// TODO if there's a real direct child, hide the deeper
		// subcommands that are under it

		// +1 for the slash
		dropPrefix := len(r.last().pkg) + 1

		wTab := tabwriter.NewWriter(w, 0, 0, 4, ' ', 0)
		for _, c := range subs {
			desc := ""
			if d, ok := c.cmd.(DescriptionGetter); ok {
				desc = d.GetDescription()
			}
			subcommand := strings.Replace(c.pkg[dropPrefix:], "/", " ", -1)
			if desc != "" {
				desc = "\t" + desc
			}
			fmt.Fprintf(wTab, "  %s%s\n", subcommand, desc)
		}
		wTab.Flush()
	}
}
