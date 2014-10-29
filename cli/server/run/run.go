package run

import (
	"sync"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/control"
	"bazil.org/bazil/server"
	"bazil.org/bazil/server/http"
	"github.com/cespare/gomaxprocs"
)

type runCommand struct {
	subcommands.Description
}

func (cmd *runCommand) Run() error {
	gomaxprocs.SetToNumCPU()
	app, err := server.New(clibazil.Bazil.Config.DataDir.String())
	if err != nil {
		return err
	}
	defer app.Close()

	errCh := make(chan error)
	var wg sync.WaitGroup

	w, err := http.New(app)
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer w.Close()
		errCh <- w.Serve()
	}()

	c, err := control.New(app)
	if err != nil {
		return err
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer c.Close()
		errCh <- c.Serve()
	}()

	wg.Wait()
	// We only care about the first error; the rest are likely to be
	// about closed listeners.
	return <-errCh
}

var run = runCommand{
	Description: "run bazil server",
}

func init() {
	subcommands.Register(&run)
}
