package create

import (
	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/server"
)

type createCommand struct {
	subcommands.Description
}

func (c *createCommand) Run() error {
	dataDir := clibazil.Bazil.Config.DataDir.String()
	app, err := server.New(dataDir)
	if err != nil {
		return err
	}

	app.Close()
	return nil
}

var create = createCommand{
	Description: "create a new data directory",
}

func init() {
	subcommands.Register(&create)
}
