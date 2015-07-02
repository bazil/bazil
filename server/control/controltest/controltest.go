package controltest

import (
	"sync"
	"testing"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control"
)

func ListenAndServe(t testing.TB, wg *sync.WaitGroup, app *server.App) *control.Control {
	c, err := control.New(app)
	if err != nil {
		t.Fatalf("control socket cannot listen: %v", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// https://github.com/golang/go/issues/4373 makes it too hard to
		// filter out innocent errors, so we throw them all out.
		_ = c.Serve()
	}()
	return c
}
