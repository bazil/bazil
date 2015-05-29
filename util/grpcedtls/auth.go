package grpcedtls

import (
	"crypto/tls"
	"errors"
	"net"
	"time"

	"bazil.org/bazil/util/edtls"
	"github.com/agl/ed25519"
	"golang.org/x/net/context"
	"google.golang.org/grpc/credentials"
)

var (
	errMissingTLSConfig = errors.New("missing TLS configuration")
)

type Authenticator struct {
	Config  func() (*tls.Config, error)
	PeerPub *[ed25519.PublicKeySize]byte
}

var _ credentials.TransportAuthenticator = (*Authenticator)(nil)

func (a *Authenticator) GetRequestMetadata(ctx context.Context) (map[string]string, error) {
	return nil, nil
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

func (a *Authenticator) ClientHandshake(addr string, rawConn net.Conn, timeout time.Duration) (net.Conn, error) {
	if a.Config == nil {
		return nil, errMissingTLSConfig
	}
	conf, err := a.Config()
	if err != nil {
		return nil, err
	}
	// We do our own verification, with edtls.
	conf.InsecureSkipVerify = true
	return edtls.NewClient(rawConn, conf, a.PeerPub)
}

func (a *Authenticator) ServerHandshake(conn net.Conn) (net.Conn, error) {
	if a.Config == nil {
		return nil, errMissingTLSConfig
	}
	tlsConf, err := a.Config()
	if err != nil {
		conn.Close()
		return nil, err
	}
	tconn := tls.Server(conn, tlsConf)
	if err := tconn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}
	return tconn, nil
}

func (a *Authenticator) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "TODO",
		SecurityVersion:  "TODO",
	}
}
