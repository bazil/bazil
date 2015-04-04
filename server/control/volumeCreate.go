package control

import (
	"bazil.org/bazil/db"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) VolumeCreate(ctx context.Context, req *wire.VolumeCreateRequest) (*wire.VolumeCreateResponse, error) {
	volumeCreate := func(tx *db.Tx) error {
		sharingKey, err := tx.SharingKeys().Get(req.SharingKeyName)
		if err != nil {
			return err
		}
		if _, err := tx.Volumes().Create(req.VolumeName, req.Backend, sharingKey); err != nil {
			return err
		}
		return nil
	}
	if err := c.app.DB.Update(volumeCreate); err != nil {
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
	return &wire.VolumeCreateResponse{}, nil
}
