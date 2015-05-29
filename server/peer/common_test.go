package peer_test

import (
	"sync"
	"testing"

	"bazil.org/bazil/server"
	"bazil.org/bazil/server/http"
)

func serveHTTP(t testing.TB, wg *sync.WaitGroup, app *server.App) *http.Web {
	web, err := http.New(app)
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
