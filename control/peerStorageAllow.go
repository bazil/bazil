package control

import (
	"log"

	"bazil.org/bazil/control/wire"
	"bazil.org/bazil/server"
	"bazil.org/bazil/tokens"
	"github.com/agl/ed25519"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) PeerStorageAllow(ctx context.Context, req *wire.PeerStorageAllowRequest) (*wire.PeerStorageAllowResponse, error) {
	if len(req.Pub) != ed25519.PublicKeySize {
		return nil, grpc.Errorf(codes.InvalidArgument, "peer public key must be exactly %d bytes", ed25519.PublicKeySize)
	}

	var pub [ed25519.PublicKeySize]byte
	copy(pub[:], req.Pub)
	_, err := c.app.GetPeer(&pub)
	if err == server.ErrPeerNotFound {
		return nil, grpc.Errorf(codes.InvalidArgument, "peer not found")
	}
	if err != nil {
		log.Printf("db error: getting peer public key %x: %v", pub[:], err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}

	for backend, _ := range req.Backends.Backends {
		if err := c.app.ValidateKV(backend); err != nil {
			return nil, grpc.Errorf(codes.InvalidArgument, "invalid backend: %q", backend)
		}
	}

	// TODO don't overwrite previous value, merge

	buf, err := proto.Marshal(req.Backends)
	if err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, "cannot re-marshal backends")
	}

	update := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketPeerStorage))
		if err := bucket.Put(pub[:], buf); err != nil {
			return err
		}
		return nil
	}
	if err := c.app.DB.Update(update); err != nil {
		log.Printf("db error: setting peer addr: %v", err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}

	return &wire.PeerStorageAllowResponse{}, nil
}
