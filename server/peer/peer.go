package peer

import (
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server"
	"google.golang.org/grpc"
)

type peers struct {
	app *server.App
}

func New(app *server.App) *grpc.Server {
	srv := grpc.NewServer()
	rpc := &peers{app: app}
	wire.RegisterPeerServer(srv, rpc)
	return srv
}
