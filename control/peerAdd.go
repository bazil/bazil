package control

import (
	"log"

	"bazil.org/bazil/control/wire"
	"github.com/agl/ed25519"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) PeerAdd(ctx context.Context, req *wire.PeerAddRequest) (*wire.PeerAddResponse, error) {
	if len(req.Pub) != ed25519.PublicKeySize {
		return nil, grpc.Errorf(codes.InvalidArgument, "peer public key must be exactly %d bytes", ed25519.PublicKeySize)
	}

	var pub [ed25519.PublicKeySize]byte
	copy(pub[:], req.Pub)
	_, err := c.app.MakePeer(&pub)
	if err != nil {
		log.Printf("db update error: put public key %x: %v", pub[:], err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerAddResponse{}, nil
}
