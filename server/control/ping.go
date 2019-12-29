package control

import (
	"context"

	"bazil.org/bazil/server/control/wire"
)

func (c controlRPC) Ping(ctx context.Context, req *wire.PingRequest) (*wire.PingResponse, error) {
	return &wire.PingResponse{}, nil
}
