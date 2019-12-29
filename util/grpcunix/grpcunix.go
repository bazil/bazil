package grpcunix

import (
	"google.golang.org/grpc"
)

func Dial(path string, opts ...grpc.DialOption) (*grpc.ClientConn, error) {
	opts = append(opts,
		// UNIX access controls are our security
		grpc.WithInsecure(),
	)
	return grpc.Dial("unix:"+path, opts...)
}
