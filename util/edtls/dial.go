package edtls

import (
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"

	"github.com/agl/ed25519"
)

var (
	// ErrNotEdTLS is returned if the TLS peer does not support edtls.
	ErrNotEdTLS = errors.New("peer does not support edtls")
)

// WrongPublicKeyError is returned if the server public key did not
// match.
type WrongPublicKeyError struct {
	Pub *[ed25519.PublicKeySize]byte
}

var _ error = (*WrongPublicKeyError)(nil)

func (e *WrongPublicKeyError) Error() string {
	return fmt.Sprintf("wrong public key: %x", e.Pub[:])
}

var defaultDialer net.Dialer

// Dial connects to a remote peer, that much have the given edtls public key.
func Dial(dialer *net.Dialer, network, addr string, config *tls.Config, peerPub *[ed25519.PublicKeySize]byte) (*tls.Conn, error) {
	if dialer == nil {
		dialer = &defaultDialer
	}
	c, err := tls.DialWithDialer(dialer, network, addr, config)
	if err != nil {
		return nil, err
	}
	if err := c.Handshake(); err != nil {
		_ = c.Close()
		return nil, err
	}
	s := c.ConnectionState()
	if len(s.PeerCertificates) == 0 {
		// servers are not supposed to be able to do that
		_ = c.Close()
		return nil, ErrNotEdTLS
	}
	pub, ok := Verify(s.PeerCertificates[0])
	if !ok {
		_ = c.Close()
		return nil, ErrNotEdTLS
	}
	if subtle.ConstantTimeCompare(pub[:], peerPub[:]) != 1 {
		_ = c.Close()
		return nil, &WrongPublicKeyError{Pub: pub}
	}
	return c, nil
}
