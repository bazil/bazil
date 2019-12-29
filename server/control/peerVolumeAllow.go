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

func (c controlRPC) PeerVolumeAllow(ctx context.Context, req *wire.PeerVolumeAllowRequest) (*wire.PeerVolumeAllowResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
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
			return nil, status.Errorf(codes.InvalidArgument, "peer not found")
		}
		log.Printf("db error: allowing peer volume: %v", err)
		return nil, status.Errorf(codes.Internal, "database error")
	}
	return &wire.PeerVolumeAllowResponse{}, nil
}
