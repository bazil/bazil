package peer

import (
	"bazil.org/bazil/peer"
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server"
	"bazil.org/bazil/util/grpcedtls"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (p *peers) auth(ctx context.Context) (*peer.Peer, error) {
	pub, ok := grpcedtls.FromContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.Unauthenticated, "unauthenticated")
	}
	peer, err := p.app.GetPeer(pub)
	if err == server.ErrPeerNotFound {
		return nil, grpc.Errorf(codes.PermissionDenied, "permission denied")
	}
	if err != nil {
		return nil, err
	}
	return peer, nil
}

type peers struct {
	app *server.App
}

func New(app *server.App) *grpc.Server {
	auth := &grpcedtls.Authenticator{
		Config: app.GetTLSConfig,
		// TODO Lookup:
	}
	srv := grpc.NewServer(
		grpc.WithServerTransportAuthenticator(auth),
	)
	rpc := &peers{app: app}
	wire.RegisterPeerServer(srv, rpc)
	return srv
}
