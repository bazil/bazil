package control

import (
	"context"

	"bazil.org/bazil/server/control/wire"
)

func (c controlRPC) VolumeMount(ctx context.Context, req *wire.VolumeMountRequest) (*wire.VolumeMountResponse, error) {
	ref, err := c.app.GetVolumeByName(req.VolumeName)
	if err != nil {
		return nil, err
	}
	defer ref.Close()
	if err := ref.Mount(req.Mountpoint); err != nil {
		return nil, err
	}
	return &wire.VolumeMountResponse{}, nil
}
