package version

import (
	"fmt"

	"bazil.org/bazil/cliutil/subcommands"
	v "bazil.org/bazil/version"
)

type versionCommand struct {
	subcommands.Description
}

func (c *versionCommand) Run() error {
	fmt.Println(v.Version)
	return nil
}

var version = versionCommand{
	Description: "show version number",
}

func init() {
	subcommands.Register(&version)
}
