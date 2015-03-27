package peer

import (
	"log"

	"bazil.org/bazil/kv"
	"bazil.org/bazil/peer/wire"
	"bazil.org/bazil/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (p *peers) ObjectGet(req *wire.ObjectGetRequest, stream wire.Peer_ObjectGetServer) error {
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

	buf, err := store.Get(req.Key)
	if err != nil {
		if _, ok := err.(kv.NotFoundError); ok {
			return grpc.Errorf(codes.NotFound, err.Error())
		}
		// TODO safe errors
		log.Printf("kv error: getting key for peer: %v", err)
		return grpc.Errorf(codes.Internal, "internal error")
	}

	const chunkSize = 4 * 1024 * 1024
	var chunk []byte
	for len(buf) > 0 {
		size := chunkSize
		if size > len(buf) {
			size = len(buf)
		}
		chunk, buf = buf[:size], buf[size:]
		if err := stream.Send(&wire.ObjectGetResponse{Data: chunk}); err != nil {
			return err
		}
	}
	return nil
}
