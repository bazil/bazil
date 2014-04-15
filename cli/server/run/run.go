package run

import (
	"flag"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/control"
	"bazil.org/bazil/server"
)

type runCommand struct {
	subcommands.Description
	flag.FlagSet
}

func (cmd *runCommand) Run() error {
	app, err := server.New(clibazil.Bazil.Config.DataDir.String())
	if err != nil {
		return err
	}
	defer app.Close()
	c := control.New(app)
	return c.ListenAndServe()
}

var run = runCommand{
	Description: "run bazil server",
}

func init() {
	subcommands.Register(&run)
}
