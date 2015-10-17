package run

import (
	"flag"
	"log"
	"net"
	"sync"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/flagx"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/server"
	"bazil.org/bazil/server/control"
	"bazil.org/bazil/server/http"
	"bazil.org/bazil/tokens"
	"bazil.org/bazil/util/trylisten"
	"github.com/cespare/gomaxprocs"
)

type tcpAddr struct {
	flagx.TCPAddr
}

func (a *tcpAddr) Set(value string) error {
	if err := a.TCPAddr.Set(value); err != nil {
		return err
	}
	run.Config.AnyPort = false
	return nil
}

type runCommand struct {
	subcommands.Description
	flag.FlagSet
	Config struct {
		Addr    tcpAddr
		AnyPort bool
	}
}

func (cmd *runCommand) Run() error {
	gomaxprocs.SetToNumCPU()
	var options []server.AppOption
	if clibazil.Bazil.Config.Debug {
		options = append(options, server.Debug(clibazil.Bazil.Log.Event))
	}
	app, err := server.New(clibazil.Bazil.Config.DataDir.String(), options...)
	if err != nil {
		return err
	}
	defer app.Close()

	errCh := make(chan error)
	var wg sync.WaitGroup

	listenTCP := net.ListenTCP
	if cmd.Config.AnyPort {
		listenTCP = trylisten.ListenTCP
	}
	l, err := listenTCP("tcp", cmd.Config.Addr.Addr)
	if err != nil {
		return err
	}

	w, err := http.New(app, l)
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

	log.Printf("Listening on %s", w.Addr())

	wg.Wait()
	// We only care about the first error; the rest are likely to be
	// about closed listeners.
	return <-errCh
}

var run = runCommand{
	Description: "run bazil server",
}

func init() {
	run.Config.Addr.Addr = &net.TCPAddr{
		Port: tokens.TCPPortHTTP,
	}
	run.Var(&run.Config.Addr, "addr", "TCP address to listen on, also sets -any-port=false")
	run.BoolVar(&run.Config.AnyPort, "any-port", true, "find a free port if port was taken")
	subcommands.Register(&run)
}
