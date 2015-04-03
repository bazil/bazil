package control

import (
	"bytes"
	"log"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) PeerAdd(ctx context.Context, req *wire.PeerAddRequest) (*wire.PeerAddResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}
	if bytes.Equal(pub[:], c.app.Keys.Sign.Pub[:]) {
		return nil, grpc.Errorf(codes.InvalidArgument, "cannot add self as peer")
	}

	makePeer := func(tx *db.Tx) error {
		if _, err := tx.Peers().Make(&pub); err != nil {
			return err
		}
		return nil
	}
	if err := c.app.DB.Update(makePeer); err != nil {
		log.Printf("db update error: put public key %x: %v", pub[:], err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerAddResponse{}, nil
}
