package add

import (
	"flag"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/positional"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/control/wire"
	"golang.org/x/net/context"
)

type addCommand struct {
	subcommands.Description
	flag.FlagSet
	Config struct {
		Sharing string
	}
	Arguments struct {
		VolumeName string
		Name       string
		positional.Optional
		Storage string
	}
}

func (cmd *addCommand) Run() error {
	storage := cmd.Arguments.Storage
	if storage == "" {
		storage = cmd.Arguments.Name
	}
	req := &wire.VolumeStorageAddRequest{
		VolumeName:     cmd.Arguments.VolumeName,
		Name:           cmd.Arguments.Name,
		Backend:        storage,
		SharingKeyName: cmd.Config.Sharing,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.VolumeStorageAdd(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var add = addCommand{
	Description: "add storage to a volume",
}

func init() {
	add.StringVar(&add.Config.Sharing, "sharing", "default", "sharing group to encrypt content for")
	subcommands.Register(&add)
}
