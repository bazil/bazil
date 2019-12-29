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

func (c controlRPC) PeerLocationSet(ctx context.Context, req *wire.PeerLocationSetRequest) (*wire.PeerLocationSetResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}

	setLoc := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(&pub)
		if err != nil {
			return err
		}
		return p.Locations().Set(req.Netloc)
	}
	if err := c.app.DB.Update(setLoc); err != nil {
		if err == db.ErrPeerNotFound {
			return nil, status.Errorf(codes.InvalidArgument, "peer not found")
		}
		log.Printf("db error: setting peer addr: %v", err)
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerLocationSetResponse{}, nil
}
