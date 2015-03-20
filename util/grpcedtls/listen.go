package grpcedtls

import (
	"crypto/tls"
	"net"
)

type listener struct {
	net.Listener
	config func() (*tls.Config, error)
}

var _ net.Listener = (*listener)(nil)

func (l *listener) Accept() (net.Conn, error) {
	if l.config == nil {
		return nil, errMissingTLSConfig
	}

	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	tlsConf, err := l.config()
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
