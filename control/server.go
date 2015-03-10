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
	app      *server.App
	listener net.Listener
}

// New creates a control socket to listen for administrative commands.
// Caller is expected to call Control.Serve to actually process
// incoming requests and Control.Close to clean up.
func New(app *server.App) (*Control, error) {
	socketPath := filepath.Join(app.DataDir, "control")
	// because app holds lock, this is safe
	err := os.Remove(socketPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	l, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	c := &Control{
		app:      app,
		listener: l,
	}
	return c, nil
}

func (c *Control) Close() {
	_ = c.listener.Close()
}

func (c *Control) Serve() error {
	srv := grpc.NewServer()
	wire.RegisterControlServer(srv, controlRPC{c})
	return srv.Serve(c.listener)
}

type controlRPC struct {
	*Control
}

var _ wire.ControlServer = controlRPC{}
