package kvfiles

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"

	"bazil.org/bazil/kv"
	"golang.org/x/net/context"
)

type KVFiles struct {
	path string
}

var _ kv.KV = (*KVFiles)(nil)

func (k *KVFiles) Put(ctx context.Context, key, value []byte) error {
	tmp, err := ioutil.TempFile(k.path, "put-")
	if err != nil {
		return err
	}
	defer func() {
		// silence errcheck
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	_, err = tmp.Write(value)
	if err != nil {
		return err
	}
	path := path.Join(k.path, hex.EncodeToString(key)+".data")
	err = os.Link(tmp.Name(), path)
	if err != nil {
		// EEXIST is safe to ignore here, that just means we
		// successfully de-duplicated content
		if !os.IsExist(err) {
			return err
		}
	}
	return nil
}

func (k *KVFiles) Get(ctx context.Context, key []byte) ([]byte, error) {
	safe := hex.EncodeToString(key)
	path := path.Join(k.path, safe+".data")
	data, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, kv.NotFoundError{
				Key: key,
			}
		}
		// no specific error to return, so just pass it through
		return nil, err
	}
	return data, nil
}

func Open(path string) (*KVFiles, error) {
	return &KVFiles{
		path: path,
	}, nil
}

func Create(path string) error {
	// this may later include more

	err := os.Mkdir(path, 0700)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}
