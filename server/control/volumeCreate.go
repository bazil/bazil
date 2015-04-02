package control

import (
	"bazil.org/bazil/fs"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

func (c controlRPC) VolumeCreate(ctx context.Context, req *wire.VolumeCreateRequest) (*wire.VolumeCreateResponse, error) {
	if err := fs.Create(c.app.DB.DB, req.VolumeName); err != nil {
		return nil, err
	}
	return &wire.VolumeCreateResponse{}, nil
}
