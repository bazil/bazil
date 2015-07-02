package control_test

import (
	"fmt"
	"testing"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func controlListenAndServe(t testing.TB, app *server.App) (stop func()) {
	c, err := control.New(app)
	if err != nil {
		t.Fatalf("control socket cannot listen: %v", err)
	}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- c.Serve()
	}()

	quit := make(chan struct{})
	go func() {
		select {
		case <-quit:
			c.Close()
			_ = <-serveErr

		case err := <-serveErr:
			if err != nil {
				t.Errorf("control socket serve: %v", err)
			}
		}
	}()

	return func() {
		close(quit)
	}
}

func checkRPCError(err error, code codes.Code, message string) error {
	if g, e := grpc.Code(err), code; g != e {
		return fmt.Errorf("wrong grpc error code: %v != %v", g, e)
	}
	// TODO https://github.com/grpc/grpc-go/issues/110
	if g, e := err.Error(), fmt.Sprintf("rpc error: code = %d desc = %q", grpc.Code(err), message); g != e {
		return fmt.Errorf("wrong error message: %v != %v", g, e)
	}
	return nil
}
