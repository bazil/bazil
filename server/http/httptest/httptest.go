package httptest

import (
	"net"
	"sync"
	"testing"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/http"
)

func ServeHTTP(t testing.TB, wg *sync.WaitGroup, app *server.App) *http.Web {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("cannot listen: %v", err)
	}
	web, err := http.New(app, l)
	if err != nil {
		t.Fatalf("cannot listen: %v", err)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		// https://github.com/golang/go/issues/4373 makes it too hard to
		// filter out innocent errors, so we throw them all out.
		_ = web.Serve()
	}()
	return web
}
