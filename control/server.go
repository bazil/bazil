package control

import (
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bazil.org/bazil/server"
)

type Control struct {
	app *server.App
	mux *http.ServeMux
}

func empty(w http.ResponseWriter, req *http.Request) {}

func New(app *server.App) *Control {
	c := &Control{
		app: app,
		mux: http.NewServeMux(),
	}

	// for ping
	c.mux.HandleFunc("/control/", empty)
	c.mux.HandleFunc("/control/volumeCreate", c.volumeCreate)
	c.mux.HandleFunc("/control/volumeMount", c.volumeMount)
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

	srv := http.Server{
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 60 * time.Second,
		Handler:      c.mux,
	}
	return srv.Serve(l)
}
