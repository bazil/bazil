package allow

import (
	"context"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
)

type allowCommand struct {
	subcommands.Description
	Arguments struct {
		PubKey     peer.PublicKey
		VolumeName string
	}
}

func (cmd *allowCommand) Run() error {
	req := &wire.PeerVolumeAllowRequest{
		Pub:        cmd.Arguments.PubKey[:],
		VolumeName: cmd.Arguments.VolumeName,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.PeerVolumeAllow(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var allow = allowCommand{
	Description: "allow a peer to use a volume",
}

func init() {
	subcommands.Register(&allow)
}
