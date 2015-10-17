package http

import (
	"net"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/peer"
)

type Web struct {
	app      *server.App
	listener net.Listener
}

func New(app *server.App, listener net.Listener) (*Web, error) {
	w := &Web{
		app:      app,
		listener: listener,
	}
	return w, nil
}

func (w *Web) Close() {
	_ = w.listener.Close()
}

func (w *Web) Serve() error {
	// TODO serve HTTPS for non-gRPC clients
	// https://github.com/grpc/grpc-go/issues/75
	srv := peer.New(w.app)
	return srv.Serve(w.listener)
}

func (w *Web) Addr() net.Addr {
	return w.listener.Addr()
}
