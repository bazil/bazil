package control

import (
	"context"
	"log"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c controlRPC) PeerStorageAllow(ctx context.Context, req *wire.PeerStorageAllowRequest) (*wire.PeerStorageAllowResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}

	if err := c.app.ValidateKV(req.Backend); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid backend: %q", req.Backend)
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
			return nil, status.Errorf(codes.InvalidArgument, "peer not found")
		}
		log.Printf("db error: allowing peer storage: %v", err)
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerStorageAllowResponse{}, nil
}
