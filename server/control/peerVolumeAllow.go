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

func (c controlRPC) PeerVolumeAllow(ctx context.Context, req *wire.PeerVolumeAllowRequest) (*wire.PeerVolumeAllowResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}

	allowVolume := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(&pub)
		if err != nil {
			return err
		}
		v, err := tx.Volumes().GetByName(req.VolumeName)
		if err != nil {
			return err
		}
		return p.Volumes().Allow(v)
	}
	if err := c.app.DB.Update(allowVolume); err != nil {
		if err == db.ErrPeerNotFound {
			return nil, grpc.Errorf(codes.InvalidArgument, "peer not found")
		}
		log.Printf("db error: allowing peer volume: %v", err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerVolumeAllowResponse{}, nil
}
