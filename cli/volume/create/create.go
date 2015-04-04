package create

import (
	"flag"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

type createCommand struct {
	subcommands.Description
	flag.FlagSet
	Config struct {
		Backend string
		Sharing string
	}
	Arguments struct {
		VolumeName string
	}
}

func (cmd *createCommand) Run() error {
	req := &wire.VolumeCreateRequest{
		VolumeName:     cmd.Arguments.VolumeName,
		Backend:        cmd.Config.Backend,
		SharingKeyName: cmd.Config.Sharing,
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
	create.StringVar(&create.Config.Backend, "backend", "local", "storage backend to use")
	create.StringVar(&create.Config.Sharing, "sharing", "default", "sharing group to encrypt content for")
	subcommands.Register(&create)
}
