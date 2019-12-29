package control

import (
	"context"

	"bazil.org/bazil/server/control/wire"
)

func (c controlRPC) PublicKeyGet(ctx context.Context, req *wire.PublicKeyGetRequest) (*wire.PublicKeyGetResponse, error) {
	resp := &wire.PublicKeyGetResponse{
		Pub: c.app.Keys.Sign.Pub[:],
	}
	return resp, nil
}
