package control

import (
	"context"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	wirepeer "bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server/control/wire"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) VolumeConnect(ctx context.Context, req *wire.VolumeConnectRequest) (*wire.VolumeConnectResponse, error) {
	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}

	if err := c.app.ValidateKV(req.Backend); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid backend: %q", req.Backend)
	}

	client, err := c.app.DialPeer(&pub)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	presp, err := client.VolumeConnect(ctx, &wirepeer.VolumeConnectRequest{
		VolumeName: req.VolumeName,
	})
	if err != nil {
		return nil, err
	}
	var volID db.VolumeID
	if err := volID.UnmarshalBinary(presp.VolumeID); err != nil {
		return nil, err
	}

	volumeConnect := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Get(req.SharingKeyName)
		if err != nil {
			return err
		}
		v, err := tx.Volumes().Add(req.LocalVolumeName, &volID, req.Backend, sharingKey)
		if err != nil {
			return err
		}

		p, err := tx.Peers().Get(&pub)
		if err != nil {
			return err
		}
		if err := p.Volumes().Allow(v); err != nil {
			return err
		}

		return nil
	}
	if err := c.app.DB.Update(volumeConnect); err != nil {
		switch err {
		case db.ErrVolNameInvalid:
			return nil, grpc.Errorf(codes.InvalidArgument, "%v", err)
		case db.ErrVolNameExist:
			return nil, grpc.Errorf(codes.AlreadyExists, "%v", err)
		case db.ErrSharingKeyNameInvalid:
			return nil, grpc.Errorf(codes.InvalidArgument, "%v", err)
		case db.ErrSharingKeyNotFound:
			return nil, grpc.Errorf(codes.FailedPrecondition, "%v", err)
		}
		return nil, err
	}
	return &wire.VolumeConnectResponse{}, nil
}
