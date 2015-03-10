package run

import (
	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/control"
	"bazil.org/bazil/server"
	"github.com/cespare/gomaxprocs"
)

type runCommand struct {
	subcommands.Description
}

func (cmd *runCommand) Run() error {
	gomaxprocs.SetToNumCPU()
	app, err := server.New(clibazil.Bazil.Config.DataDir.String())
	if err != nil {
		return err
	}
	defer app.Close()
	c, err := control.New(app)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.Serve()
}

var run = runCommand{
	Description: "run bazil server",
}

func init() {
	subcommands.Register(&run)
}
