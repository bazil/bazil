package pubkey

import (
	"os"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server"
)

type pubkeyCommand struct {
	subcommands.Description
}

func (c *pubkeyCommand) Run() error {
	app, err := server.New(clibazil.Bazil.Config.DataDir.String())
	if err != nil {
		return err
	}
	defer app.Close()

	k := peer.PublicKey(*app.Keys.Sign.Pub)
	_, err = os.Stdout.WriteString(k.String() + "\n")
	return err
}

var pubkey = pubkeyCommand{
	Description: "show public key",
}

func init() {
	subcommands.Register(&pubkey)
}
