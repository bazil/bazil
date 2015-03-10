package control

import (
	"bazil.org/bazil/control/wire"
	"bazil.org/bazil/fs"
	"golang.org/x/net/context"
)

func (c controlRPC) VolumeCreate(ctx context.Context, req *wire.VolumeCreateRequest) (*wire.VolumeCreateResponse, error) {
	if err := fs.Create(c.app.DB, req.VolumeName); err != nil {
		return nil, err
	}
	return &wire.VolumeCreateResponse{}, nil
}
