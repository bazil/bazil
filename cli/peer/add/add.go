package add

import (
	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

type addCommand struct {
	subcommands.Description
	Arguments struct {
		PubKey peer.PublicKey
	}
}

func (cmd *addCommand) Run() error {
	req := &wire.PeerAddRequest{
		Pub: cmd.Arguments.PubKey[:],
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.PeerAdd(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var add = addCommand{
	Description: "add a peer",
}

func init() {
	subcommands.Register(&add)
}
