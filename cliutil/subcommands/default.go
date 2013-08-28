package subcommands

// Default is a single global instance of Shell used for command-line
// argument parsing.
var Default Shell

// Register this command on the default Shell.
func Register(cmd interface{}) {
	Default.Register(cmd)
}

// Parse the command line using the default Shell.
//
// In typical use, cmd is the address of a registered command at the
// top of the package hierarchy where the commands reside, name is the
// name of the application running, and args is `os.Args[1:]`.
func Parse(cmd interface{}, name string, args []string) (Result, error) {
	return Default.Parse(cmd, name, args)
}
