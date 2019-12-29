package control

import (
	"context"
	"log"

	"bazil.org/bazil/db"
	"bazil.org/bazil/server/control/wire"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c controlRPC) VolumeStorageAdd(ctx context.Context, req *wire.VolumeStorageAddRequest) (*wire.VolumeStorageAddResponse, error) {
	if err := c.app.ValidateKV(req.Backend); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}

	addStorage := func(tx *db.Tx) error {
		vol, err := tx.Volumes().GetByName(req.VolumeName)
		if err != nil {
			return err
		}
		sharingKey, err := tx.SharingKeys().Get(req.SharingKeyName)
		if err != nil {
			return err
		}
		return vol.Storage().Add(req.Name, req.Backend, sharingKey)
	}
	if err := c.app.DB.Update(addStorage); err != nil {
		switch err {
		case db.ErrVolNameNotFound:
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		case db.ErrSharingKeyNameInvalid:
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		case db.ErrSharingKeyNotFound:
			return nil, status.Errorf(codes.FailedPrecondition, "%v", err)
		case db.ErrVolumeStorageExist:
			return nil, status.Errorf(codes.AlreadyExists, err.Error())
		}
		log.Printf("db update error: add storage %q: %v", req.Name, err)
		return nil, status.Errorf(codes.Internal, "Internal error")
	}
	return &wire.VolumeStorageAddResponse{}, nil
}
