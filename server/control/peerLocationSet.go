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

func (c controlRPC) PeerLocationSet(ctx context.Context, req *wire.PeerLocationSetRequest) (*wire.PeerLocationSetResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
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
			return nil, grpc.Errorf(codes.InvalidArgument, "peer not found")
		}
		log.Printf("db error: setting peer addr: %v", err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerLocationSetResponse{}, nil
}
