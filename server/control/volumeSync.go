package control

import (
	"io"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer"
	wirepeer "bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) VolumeSync(ctx context.Context, req *wire.VolumeSyncRequest) (*wire.VolumeSyncResponse, error) {
	var volID db.VolumeID
	loadVolume := func(tx *db.Tx) error {
		v, err := tx.Volumes().GetByName(req.VolumeName)
		if err != nil {
			if err == db.ErrVolNameNotFound {
				return grpc.Errorf(codes.InvalidArgument, "%v", err)
			}
			return err
		}
		v.VolumeID(&volID)
		return nil
	}
	if err := c.app.DB.View(loadVolume); err != nil {
		return nil, err
	}

	var pub peer.PublicKey
	if err := pub.UnmarshalBinary(req.Pub); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "bad peer public key: %v", err)
	}

	client, err := c.app.DialPeer(&pub)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	volIDBuf, err := volID.MarshalBinary()
	if err != nil {
		return nil, err
	}

	peerReq := &wirepeer.VolumeSyncPullRequest{
		VolumeID: volIDBuf,
		Path:     req.Path,
	}
	stream, err := client.VolumeSyncPull(ctx, peerReq)
	if err != nil {
		return nil, err
	}

	first, err := stream.Recv()
	if err != nil && err != io.EOF {
		return nil, err
	}

	switch first.Error {
	case wirepeer.VolumeSyncPullItem_SUCCESS:
		// nothing
	case wirepeer.VolumeSyncPullItem_NOT_A_DIRECTORY:
		// TODO maybe we should handle the path not being a dir, somehow
		return nil, grpc.Errorf(codes.FailedPrecondition, "path to sync is not a directory")
	default:
		return nil, grpc.Errorf(codes.FailedPrecondition, "peer gave error: %v", first.Error.String())
	}

	recv := func() ([]*wirepeer.Dirent, error) {
		if first.Children != nil {
			tmp := first.Children
			first.Children = nil
			return tmp, nil
		}
		item, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		return item.Children, nil
	}

	ref, err := c.app.GetVolume(&volID)
	if err != nil {
		return nil, err
	}
	defer ref.Close()

	if err := ref.FS().SyncReceive(ctx, req.Path, first.Peers, first.DirClock, recv); err != nil {
		return nil, err
	}

	return &wire.VolumeSyncResponse{}, nil
}
