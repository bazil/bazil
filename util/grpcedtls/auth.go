package grpcedtls

import (
	"crypto/tls"
	"errors"
	"net"

	"bazil.org/bazil/util/edtls"
	"github.com/agl/ed25519"
	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials"
)

var (
	errMissingTLSConfig = errors.New("missing TLS configuration")
	errMissingLookup    = errors.New("missing peer key lookup mechanism")
)

type Authenticator struct {
	Config func() (*tls.Config, error)
	Lookup func(network string, addr string) (network2, addr2 string, peerPub *[ed25519.PublicKeySize]byte, err error)
}

var _ credentials.TransportAuthenticator = (*Authenticator)(nil)

func (a *Authenticator) GetRequestMetadata(ctx context.Context) (map[string]string, error) {
	return nil, nil
}

func (a *Authenticator) DialWithDialer(dialer *net.Dialer, network, addr string) (_ net.Conn, err error) {
	if a.Config == nil {
		return nil, errMissingTLSConfig
	}
	conf, err := a.Config()
	if err != nil {
		return nil, err
	}
	if a.Lookup == nil {
		return nil, errMissingLookup
	}
	network, addr, peerPub, err := a.Lookup(network, addr)
	if err != nil {
		return nil, err
	}
	// We do our own verification, with edtls.
	conf.InsecureSkipVerify = true
	return edtls.Dial(dialer, network, addr, conf, peerPub)
}

func (a *Authenticator) Dial(network, addr string) (_ net.Conn, err error) {
	return a.DialWithDialer(&net.Dialer{}, network, addr)
}

func (a *Authenticator) NewListener(lis net.Listener) net.Listener {
	return &listener{
		Listener: lis,
		config:   a.Config,
	}
}

type peerKeyT int

const peerKey = peerKeyT(0)

func (a *Authenticator) NewServerConn(ctx context.Context, conn net.Conn) context.Context {
	return context.WithValue(ctx, peerKey, conn)
}

func FromContext(ctx context.Context) (pub *[ed25519.PublicKeySize]byte, ok bool) {
	v := ctx.Value(peerKey)
	if v == nil {
		return nil, false
	}
	tconn, ok := v.(*tls.Conn)
	if !ok {
		return nil, false
	}
	if err := tconn.Handshake(); err != nil {
		return nil, false
	}
	state := tconn.ConnectionState()
	if !state.HandshakeComplete {
		return nil, false
	}
	if len(state.PeerCertificates) == 0 {
		return nil, false
	}
	return edtls.Verify(state.PeerCertificates[0])
}
