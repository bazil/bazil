package control

import (
	"log"

	"bazil.org/bazil/control/wire"
	"bazil.org/bazil/tokens"
	"github.com/boltdb/bolt"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

const sharingKeySize = 32

func (c controlRPC) SharingKeyAdd(ctx context.Context, req *wire.SharingKeyAddRequest) (*wire.SharingKeyAddResponse, error) {
	if len(req.Secret) != sharingKeySize {
		return nil, grpc.Errorf(codes.InvalidArgument, "sharing key must be exactly 32 bytes")
	}
	if req.Name == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid sharing key name")
	}

	update := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketSharing))
		key := []byte(req.Name)
		if bucket.Get(key) != nil {
			return grpc.Errorf(codes.AlreadyExists, "sharing key exists already: %x", req.Name)
		}
		if err := bucket.Put(key, req.Secret); err != nil {
			return err
		}
		return nil
	}
	if err := c.app.DB.Update(update); err != nil {
		if grpc.Code(err) != codes.Unknown {
			return nil, err
		}
		log.Printf("db update error: put sharing key %q: %v", req.Name, err)
		return nil, grpc.Errorf(codes.Internal, "database error")
	}
	return &wire.SharingKeyAddResponse{}, nil
}
