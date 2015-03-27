package peer

import (
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server"
)

func (p *peers) ObjectPut(stream wire.Peer_ObjectPutServer) error {
	client, err := p.auth(stream.Context())
	if err != nil {
		return err
	}
	store, err := p.app.OpenKVForPeer(client.Pub)
	if err != nil {
		if err == server.ErrNoStorageForPeer {
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

	if err := store.Put(key, data); err != nil {
		return err
	}
	return stream.SendAndClose(&wire.ObjectPutResponse{})
}
