package peer

import (
	"bazil.org/bazil/peer/wire"
	"golang.org/x/net/context"
)

func (p *peers) Ping(ctx context.Context, req *wire.PingRequest) (*wire.PingResponse, error) {
	_, err := p.auth(ctx)
	if err != nil {
		return nil, err
	}
	return &wire.PingResponse{}, nil
}
