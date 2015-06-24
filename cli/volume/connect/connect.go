package connect

import (
	"flag"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/positional"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

type connectCommand struct {
	subcommands.Description
	flag.FlagSet
	Config struct {
		Backend string
		Sharing string
	}
	Arguments struct {
		PubKey     peer.PublicKey
		VolumeName string
		positional.Optional
		LocalVolumeName string
	}
}

func (cmd *connectCommand) Run() error {
	localVolumeName := cmd.Arguments.LocalVolumeName
	if localVolumeName == "" {
		localVolumeName = cmd.Arguments.VolumeName
	}
	req := &wire.VolumeConnectRequest{
		Pub:             cmd.Arguments.PubKey[:],
		VolumeName:      cmd.Arguments.VolumeName,
		LocalVolumeName: localVolumeName,
		Backend:         cmd.Config.Backend,
		SharingKeyName:  cmd.Config.Sharing,
	}
	ctx := context.Background()
	client, err := clibazil.Bazil.Control()
	if err != nil {
		return err
	}
	if _, err := client.VolumeConnect(ctx, req); err != nil {
		// TODO unwrap error
		return err
	}
	return nil
}

var connect = connectCommand{
	Description: "connect to a volume at a peer",
}

func init() {
	connect.StringVar(&connect.Config.Backend, "backend", "local", "storage backend to use")
	connect.StringVar(&connect.Config.Sharing, "sharing", "default", "sharing group to encrypt content for")
	subcommands.Register(&connect)
}
