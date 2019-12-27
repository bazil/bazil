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
)

type Authenticator struct {
	Config  func() (*tls.Config, error)
	PeerPub *[ed25519.PublicKeySize]byte
}

var _ credentials.TransportCredentials = (*Authenticator)(nil)

func (a *Authenticator) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return nil, nil
}

func (a *Authenticator) RequireTransportSecurity() bool {
	return true
}

type Auth struct {
	PeerPub *[ed25519.PublicKeySize]byte
}

var _ credentials.AuthInfo = (*Auth)(nil)

func (*Auth) AuthType() string { return "edtls" }

func (a *Authenticator) ClientHandshake(ctx context.Context, addr string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if a.Config == nil {
		return nil, nil, errMissingTLSConfig
	}
	conf, err := a.Config()
	if err != nil {
		return nil, nil, err
	}
	// We do our own verification, with edtls.
	conf.InsecureSkipVerify = true
	tconn, err := edtls.NewClient(rawConn, conf, a.PeerPub)
	if err != nil {
		return nil, nil, err
	}

	authInfo := &Auth{
		PeerPub: a.PeerPub,
	}
	return tconn, authInfo, nil
}

func (a *Authenticator) ServerHandshake(conn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	if a.Config == nil {
		return nil, nil, errMissingTLSConfig
	}
	tlsConf, err := a.Config()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}
	tconn := tls.Server(conn, tlsConf)
	if err := tconn.Handshake(); err != nil {
		conn.Close()
		return nil, nil, err
	}
	state := tconn.ConnectionState()
	if !state.HandshakeComplete {
		conn.Close()
		return nil, nil, errors.New("TLS handshake did not complete")
	}
	if len(state.PeerCertificates) == 0 {
		conn.Close()
		return nil, nil, errors.New("no TLS peer certificates")
	}
	pub, ok := edtls.Verify(state.PeerCertificates[0])
	if !ok {
		conn.Close()
		return nil, nil, errors.New("edtls verification failed")
	}
	authInfo := &Auth{
		PeerPub: pub,
	}
	return tconn, authInfo, nil
}

func (a *Authenticator) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "TODO",
		SecurityVersion:  "TODO",
	}
}

func (a *Authenticator) Clone() credentials.TransportCredentials {
	aa := &Authenticator{
		Config:  a.Config,
		PeerPub: a.PeerPub,
	}
	return aa
}

func (a *Authenticator) OverrideServerName(_ string) error {
	return nil
}
