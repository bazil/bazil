package control

import (
	"bazil.org/bazil/server/control/wire"
	"golang.org/x/net/context"
)

func (c controlRPC) PublicKeyGet(ctx context.Context, req *wire.PublicKeyGetRequest) (*wire.PublicKeyGetResponse, error) {
	resp := &wire.PublicKeyGetResponse{
		Pub: c.app.Keys.Sign.Pub[:],
	}
	return resp, nil
}
