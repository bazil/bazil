package http

import (
	"net"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/peer"
)

type Web struct {
	app      *server.App
	listener *net.TCPListener
}

func New(app *server.App) (*Web, error) {
	w := &Web{
		app: app,
	}

	addr := &net.TCPAddr{}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}
	w.listener = l

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
