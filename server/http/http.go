package http

import (
	"crypto/tls"
	"errors"
	"net"

	"bazil.org/bazil/server"
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
	conf, err := w.app.GetTLSConfig()
	if err != nil {
		return err
	}
	l := tls.NewListener(w.listener, conf)

	_ = l
	return errors.New("TODO actually handle the connections")
}
