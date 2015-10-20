package control_test

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func checkRPCError(err error, code codes.Code, message string) error {
	if g, e := grpc.Code(err), code; g != e {
		return fmt.Errorf("wrong grpc error code: %v != %v", g, e)
	}
	if g, e := grpc.ErrorDesc(err), message; g != e {
		return fmt.Errorf("wrong error message: %v != %v", g, e)
	}
	return nil
}
