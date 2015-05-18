package server

import (
	"io"
	"time"

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
	lookup := func(addr string) (string, *[ed25519.PublicKeySize]byte, error) {
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
			return "", nil, err
		}
		return addr, (*[ed25519.PublicKeySize]byte)(pub), nil
	}

	auth := &grpcedtls.Authenticator{
		Config: app.GetTLSConfig,
		Lookup: lookup,
	}

	// TODO never delay here.
	// https://github.com/grpc/grpc-go/blob/8ce50750fe22e967aa8b1d308b21511844674b57/clientconn.go#L85
	conn, err := grpc.Dial("placeholder.bazil.org.invalid.:443",
		grpc.WithTransportCredentials(auth),
		grpc.WithTimeout(30*time.Second),
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
