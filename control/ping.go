package control

import (
	"bazil.org/bazil/control/wire"
	"golang.org/x/net/context"
)

func (c *Control) Ping(ctx context.Context, req *wire.PingRequest) (*wire.PingResponse, error) {
	return &wire.PingResponse{}, nil
}
