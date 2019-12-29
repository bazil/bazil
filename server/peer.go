package server

import (
	"io"

	"bazil.org/bazil/db"
	"bazil.org/bazil/kv"
	"bazil.org/bazil/peer"
	wirepeer "bazil.org/bazil/peer/wire"
	"bazil.org/bazil/util/grpcedtls"
	"github.com/agl/ed25519"
	"google.golang.org/grpc"
)

func (app *App) OpenKVForPeer(pub *peer.PublicKey) (kv.KV, error) {
	var kvstore kv.KV
	open := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(pub)
		if err != nil {
			return err
		}
		s, err := p.Storage().Open(app.openStorage)
		if err != nil {
			return err
		}
		kvstore = s
		return nil
	}
	if err := app.DB.View(open); err != nil {
		return nil, err
	}
	return kvstore, nil
}

type PeerClient interface {
	wirepeer.PeerClient
	io.Closer
}

type peerClient struct {
	wirepeer.PeerClient
	conn *grpc.ClientConn
}

var _ PeerClient = (*peerClient)(nil)

func (p *peerClient) Close() error {
	return p.conn.Close()
}

func (app *App) DialPeer(pub *peer.PublicKey) (PeerClient, error) {
	var addr string
	find := func(tx *db.Tx) error {
		p, err := tx.Peers().Get(pub)
		if err != nil {
			return err
		}
		a, err := p.Locations().Get()
		if err != nil {
			return err
		}
		addr = a
		return nil
	}
	if err := app.DB.View(find); err != nil {
		return nil, err
	}

	auth := &grpcedtls.Authenticator{
		Config:  app.GetTLSConfig,
		PeerPub: (*[ed25519.PublicKeySize]byte)(pub),
	}

	// this is not a slow network operation, it just tells grpc about
	// the remote
	conn, err := grpc.Dial(addr,
		grpc.WithTransportCredentials(auth),
	)
	if err != nil {
		return nil, err
	}
	client := wirepeer.NewPeerClient(conn)
	p := &peerClient{
		PeerClient: client,
		conn:       conn,
	}
	return p, nil
}
