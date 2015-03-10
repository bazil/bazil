package control

import (
	"errors"
	"fmt"
	"log"

	"bazil.org/bazil/control/wire"
	wirefs "bazil.org/bazil/fs/wire"
	"bazil.org/bazil/tokens"
	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type storageExistsError struct {
	Name string
}

var _ error = (*storageExistsError)(nil)

func (e *storageExistsError) Error() string {
	return fmt.Sprintf("storage backend exists already: %x", e.Name)
}

// addStorage adds the given storage backend to this volume in the
// database.
//
// Active Volume instances are not notified.
//
// TODO this logic most does not belong here, but I don't want to add
// storageExistsError to a proper API at this time either.
func addStorage(db *bolt.DB, volumeName string, name string, backend string, sharingKeyName string) error {
	addStorage := func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(tokens.BucketVolName))
		key := []byte(volumeName)
		val := bucket.Get(key)
		if val == nil {
			return errors.New("volume not found")
		}
		var volConf wirefs.VolumeConfig
		if err := proto.Unmarshal(val, &volConf); err != nil {
			return err
		}

		for _, storage := range volConf.Storage {
			if storage.Name == name {
				return &storageExistsError{Name: name}
			}
		}
		storage := &wirefs.VolumeStorage{
			Name:           name,
			Backend:        backend,
			SharingKeyName: sharingKeyName,
		}
		volConf.Storage = append(volConf.Storage, storage)

		buf, err := proto.Marshal(&volConf)
		if err != nil {
			return err
		}
		if err := bucket.Put(key, buf); err != nil {
			return err
		}
		return nil
	}
	if err := db.Update(addStorage); err != nil {
		return err
	}
	return nil
}

func (c controlRPC) VolumeStorageAdd(ctx context.Context, req *wire.VolumeStorageAddRequest) (*wire.VolumeStorageAddResponse, error) {
	if err := c.app.ValidateKV(req.Backend); err != nil {
		return nil, grpc.Errorf(codes.InvalidArgument, err.Error())
	}
	if req.SharingKeyName == "" {
		return nil, grpc.Errorf(codes.InvalidArgument, "invalid sharing key name")
	}

	if err := addStorage(c.app.DB, req.VolumeName, req.Name, req.Backend, req.SharingKeyName); err != nil {
		switch err.(type) {
		case *storageExistsError:
			return nil, grpc.Errorf(codes.AlreadyExists, err.Error())
		}
		log.Printf("db update error: put storage %q: %v", req.Name, err)
		return nil, grpc.Errorf(codes.Internal, "Internal error")
	}
	return &wire.VolumeStorageAddResponse{}, nil
}
