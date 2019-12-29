package peer

import (
	"log"

	"bazil.org/bazil/db"
	"bazil.org/bazil/kv"
	"bazil.org/bazil/peer/wire"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (p *peers) ObjectGet(req *wire.ObjectGetRequest, stream wire.Peer_ObjectGetServer) error {
	pub, err := p.auth(stream.Context())
	if err != nil {
		return err
	}
	store, err := p.app.OpenKVForPeer(pub)
	if err != nil {
		if err == db.ErrNoStorageForPeer {
			return status.Errorf(codes.PermissionDenied, "%v", err)
		}
		return err
	}

	buf, err := store.Get(stream.Context(), req.Key)
	if err != nil {
		if _, ok := err.(kv.NotFoundError); ok {
			return status.Errorf(codes.NotFound, err.Error())
		}
		// TODO safe errors
		log.Printf("kv error: getting key for peer: %v", err)
		return status.Errorf(codes.Internal, "internal error")
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
