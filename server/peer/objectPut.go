package peer

import (
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"bazil.org/bazil/db"
	"bazil.org/bazil/peer/wire"
)

func (p *peers) ObjectPut(stream wire.Peer_ObjectPutServer) error {
	pub, err := p.auth(stream.Context())
	if err != nil {
		return err
	}
	store, err := p.app.OpenKVForPeer(pub)
	if err != nil {
		if err == db.ErrNoStorageForPeer {
			return grpc.Errorf(codes.PermissionDenied, "%v", err)
		}
		return err
	}

	var key []byte
	var data []byte
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if key == nil {
			if req.Key == nil {
				return grpc.Errorf(codes.InvalidArgument, "ObjectPutRequest.Key must be set in first streamed message")
			}
			key = req.Key
		}
		data = append(data, req.Data...)
	}

	if err := store.Put(stream.Context(), key, data); err != nil {
		return err
	}
	return stream.SendAndClose(&wire.ObjectPutResponse{})
}
