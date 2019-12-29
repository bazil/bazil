package control_test

import (
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func checkRPCError(err error, code codes.Code, message string) error {
	if g, e := status.Code(err), code; g != e {
		return fmt.Errorf("wrong grpc error code: %v != %v", g, e)
	}
	if g, e := status.Convert(err).Message(), message; g != e {
		return fmt.Errorf("wrong error message: %v != %v", g, e)
	}
	return nil
}
