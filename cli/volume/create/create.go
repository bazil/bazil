package create

import (
	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

type createCommand struct {
	subcommands.Description
	Arguments struct {
		VolumeName string
	}
}

func (cmd *createCommand) Run() error {
	req := &wire.VolumeCreateRequest{
		VolumeName: cmd.Arguments.VolumeName,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.VolumeCreate(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var create = createCommand{
	Description: "create a new volume",
}

func init() {
	subcommands.Register(&create)
}
