package peer

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer/wire"
)

func (p *peers) VolumeConnect(ctx context.Context, req *wire.VolumeConnectRequest) (*wire.VolumeConnectResponse, error) {
	pub, err := p.auth(ctx)
	if err != nil {
		return nil, err
	}
	var volID db.VolumeID
	view := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(pub)
		if err != nil {
			return err
		}
		vol, err := tx.Volumes().GetByName(req.VolumeName)

		// do not leak names peer has no access to; not found gets the
		// same error as not allowed
		if (err == nil && !p.Volumes().IsAllowed(vol)) ||
			err == db.ErrVolNameNotFound {
			err = grpc.Errorf(codes.PermissionDenied, "peer is not authorized for that volume")
		}
		if err != nil {
			return err
		}
		vol.VolumeID(&volID)
		return nil
	}
	if err := p.app.DB.View(view); err != nil {
		return nil, err
	}

	resp := &wire.VolumeConnectResponse{
		VolumeID: volID[:],
	}
	return resp, nil
}
