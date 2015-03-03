package control

import (
	"net"
	"os"
	"path/filepath"

	"bazil.org/bazil/control/wire"
	"bazil.org/bazil/server"
	"google.golang.org/grpc"
)

type Control struct {
	app *server.App
}

var _ wire.ControlServer = (*Control)(nil)

func New(app *server.App) *Control {
	c := &Control{
		app: app,
	}
	return c
}

func (c *Control) ListenAndServe() error {
	socketPath := filepath.Join(c.app.DataDir, "control")
	// because app holds lock, this is safe
	err := os.Remove(socketPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	defer l.Close()

	srv := grpc.NewServer()
	wire.RegisterControlServer(srv, c)
	return srv.Serve(l)
}
