package mount

import (
	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/flagx"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

type mountCommand struct {
	subcommands.Description
	Arguments struct {
		VolumeName string
		Mountpoint flagx.AbsPath
	}
}

func (cmd *mountCommand) Run() error {
	req := &wire.VolumeMountRequest{
		VolumeName: cmd.Arguments.VolumeName,
		Mountpoint: cmd.Arguments.Mountpoint.String(),
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.VolumeMount(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var mount = mountCommand{
	Description: "mount a volume",
}

func init() {
	subcommands.Register(&mount)
}
