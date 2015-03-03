package grpcunix

import (
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

type unixDialer struct{}

var _ credentials.TransportAuthenticator = unixDialer{}

func (u unixDialer) Dial(network, addr string) (net.Conn, error) {
	return u.DialWithDialer(&net.Dialer{}, network, addr)
}

func (unixDialer) DialWithDialer(dialer *net.Dialer, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if port != "kludge" {
		panic("unix domain port kludge missing")
	}
	// TODO avoid retries on connection refused
	return dialer.Dial("unix", host)
}

func (unixDialer) NewListener(lis net.Listener) net.Listener {
	return lis
}

func (unixDialer) GetRequestMetadata(ctx context.Context) (map[string]string, error) {
	return nil, nil
}

func Dial(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts, grpc.WithTransportCredentials(unixDialer{}))
	// https://github.com/grpc/grpc-go/issues/73
	addr := net.JoinHostPort(path, "kludge")
	return grpc.Dial(addr, opts...)
}
