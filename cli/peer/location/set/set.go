package set

import (
	"context"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
)

type setCommand struct {
	subcommands.Description
	Arguments struct {
		PubKey peer.PublicKey
		Addr   string `positional:"metavar=HOST:PORT"`
	}
}

func (cmd *setCommand) Run() error {
	req := &wire.PeerLocationSetRequest{
		Pub:    cmd.Arguments.PubKey[:],
		Netloc: cmd.Arguments.Addr,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.PeerLocationSet(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var set = setCommand{
	Description: "set network location for peer",
}

func init() {
	subcommands.Register(&set)
}
