package control

import (
	"bytes"
	"context"
	"log"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c controlRPC) PeerAdd(ctx context.Context, req *wire.PeerAddRequest) (*wire.PeerAddResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}
	if bytes.Equal(pub[:], c.app.Keys.Sign.Pub[:]) {
		return nil, status.Errorf(codes.InvalidArgument, "cannot add self as peer")
	}

	makePeer := func(tx *db.Tx) error {
		if _, err := tx.Peers().Make(&pub); err != nil {
			return err
		}
		return nil
	}
	if err := c.app.DB.Update(makePeer); err != nil {
		log.Printf("db update error: put public key %x: %v", pub[:], err)
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerAddResponse{}, nil
}
