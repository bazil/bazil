package control

import (
	"log"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) PeerStorageAllow(ctx context.Context, req *wire.PeerStorageAllowRequest) (*wire.PeerStorageAllowResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}

	if err := c.app.ValidateKV(req.Backend); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid backend: %q", req.Backend)
	}

	allowStorage := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(&pub)
		if err != nil {
			return err
		}
		return p.Storage().Allow(req.Backend)
	}
	if err := c.app.DB.Update(allowStorage); err != nil {
		if err == db.ErrPeerNotFound {
			return nil, grpc.Errorf(codes.InvalidArgument, "peer not found")
		}
		log.Printf("db error: allowing peer storage: %v", err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerStorageAllowResponse{}, nil
}
