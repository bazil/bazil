package pubkey

import (
	"os"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

type pubkeyCommand struct {
	subcommands.Description
}

func (c *pubkeyCommand) Run() error {
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	resp, err := client.PublicKeyGet(ctx, &wire.PublicKeyGetRequest{})
	if err != nil {
		return err
	}

	var k peer.PublicKey
	if err := k.UnmarshalBinary(resp.Pub); err != nil {
		return err
	}
	_, err = os.Stdout.WriteString(k.String() + "\n")
	return err
}

var pubkey = pubkeyCommand{
	Description: "show public key",
}

func init() {
	subcommands.Register(&pubkey)
}
