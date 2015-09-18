package grpcunix

import (
	"net"
	"time"

	"google.golang.org/grpc"
)

func dial(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}

func Dial(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts,
		grpc.WithDialer(dial),
		// UNIX access controls are our security
		grpc.WithInsecure(),
	)
	return grpc.Dial(path, opts...)
}
