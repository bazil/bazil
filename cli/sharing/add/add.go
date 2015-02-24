package add

import (
	"errors"
	"io/ioutil"
	"os"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/control/wire"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/context"
)

type addCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		Name string
	}
}

func (cmd *addCommand) Run() error {
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("refusing to read secret from a terminal")
	}
	secret, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return err
	}

	req := &wire.SharingKeyAddRequest{
		Name:   cmd.Arguments.Name,
		Secret: secret,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.SharingKeyAdd(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var add = addCommand{
	Description: "add a new sharing key",
	Synopsis:    "NAME <SECRET_FILE",
}

func init() {
	subcommands.Register(&add)
}
