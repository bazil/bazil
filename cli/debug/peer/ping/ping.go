package ping

import (
	"fmt"
	"log"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server"
	"bazil.org/bazil/util/grpcedtls"
	"github.com/agl/ed25519"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type pingCommand struct {
	subcommands.Description
	Arguments struct {
		Addr string `positional:"metavar=HOST:PORT"`
		Pub  peer.PublicKey
	}
}

func (c *pingCommand) Run() error {
	app, err := server.New(clibazil.Bazil.Config.DataDir.String())
	if err != nil {
		return err
	}
	defer app.Close()

	pub := (*[ed25519.PublicKeySize]byte)(&c.Arguments.Pub)
	auth := &grpcedtls.Authenticator{
		Config:  app.GetTLSConfig,
		PeerPub: pub,
	}
	addr := c.Arguments.Addr
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(auth))
	if err != nil {
		return fmt.Errorf("did not connect: %v", err)
	}
	defer conn.Close()
	client := wire.NewPeerClient(conn)

	r, err := client.Ping(context.Background(), &wire.PingRequest{})
	if err != nil {
		return fmt.Errorf("could not greet: %v", err)
	}
	log.Printf("pong: %#v", r)
	return nil
}

var ping = pingCommand{
	Description: "ping a peer",
}

func init() {
	subcommands.Register(&ping)
}
