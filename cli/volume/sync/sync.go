package sync

import (
	"context"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/positional"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
)

type syncCommand struct {
	subcommands.Description
	Arguments struct {
		VolumeName string
		PubKey     peer.PublicKey
		positional.Optional
		Path string
	}
}

func (cmd *syncCommand) Run() error {
	req := &wire.VolumeSyncRequest{
		Pub:        cmd.Arguments.PubKey[:],
		VolumeName: cmd.Arguments.VolumeName,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.VolumeSync(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var sync = syncCommand{
	Description: "sync volume from peer",
}

func init() {
	subcommands.Register(&sync)
}
