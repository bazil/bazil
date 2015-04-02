package control

import (
	"log"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control/wire"
	"bazil.org/bazil/tokens"
	"github.com/agl/ed25519"
	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (c controlRPC) PeerLocationSet(ctx context.Context, req *wire.PeerLocationSetRequest) (*wire.PeerLocationSetResponse, error) {
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

	update := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketPeerAddr))
		if err := bucket.Put(pub[:], []byte(req.Netloc)); err != nil {
			return err
		}
		return nil
	}
	if err := c.app.DB.DB.Update(update); err != nil {
		log.Printf("db error: setting peer addr: %v", err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}

	return &wire.PeerLocationSetResponse{}, nil
}
