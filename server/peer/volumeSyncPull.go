package peer

import (
	"bazil.org/bazil/db"
	"bazil.org/bazil/peer/wire"
	"bazil.org/fuse"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *peers) VolumeSyncPull(req *wire.VolumeSyncPullRequest, stream wire.Peer_VolumeSyncPullServer) error {
	ctx := stream.Context()
	pub, err := p.auth(ctx)
	if err != nil {
		return err
	}
	var volID db.VolumeID
	if err := volID.UnmarshalBinary(req.VolumeID); err != nil {
		return err
	}

	view := func(tx *db.Tx) error {
		client, err := tx.Peers().Get(pub)
		if err != nil {
			return err
		}
		vol, err := tx.Volumes().GetByVolumeID(&volID)
		// do not leak names peer has no access to; not found gets the
		// same error as not allowed
		if (err == nil && !client.Volumes().IsAllowed(vol)) ||
			err == db.ErrVolumeIDNotFound {
			err = status.Errorf(codes.PermissionDenied, "peer is not authorized for that volume")
		}
		if err != nil {
			return err
		}
		return nil
	}
	if err := p.app.DB.View(view); err != nil {
		return err
	}

	v, err := p.app.GetVolume(&volID)
	if err != nil {
		return err
	}
	defer v.Close()

	if err := v.FS().SyncSend(ctx, req.Path, stream.Send); err != nil {
		if err == fuse.ENOENT {
			return status.Errorf(codes.NotFound, "not found")
		}
		return err
	}
	return nil
}
