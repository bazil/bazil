package control

import (
	"bazil.org/bazil/control/wire"
	"golang.org/x/net/context"
)

func (c controlRPC) VolumeMount(ctx context.Context, req *wire.VolumeMountRequest) (*wire.VolumeMountResponse, error) {
	if _, err := c.app.Mount(req.VolumeName, req.Mountpoint); err != nil {
		return nil, err
	}
	return &wire.VolumeMountResponse{}, nil
}
