package ping

import (
	"errors"
	"flag"
	"io/ioutil"
	"net/http"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
)

type pingCommand struct {
	subcommands.Description
	flag.FlagSet
}

func (cmd *pingCommand) Run() error {
	resp, err := clibazil.Bazil.Control.Head("http+unix://bazil/control")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		buf, _ := ioutil.ReadAll(resp.Body)
		if len(buf) == 0 {
			buf = []byte(resp.Status)
		}
		return errors.New(string(buf))
	}
	return nil
}

var ping = pingCommand{
	Description: "ping bazil server",
}

func init() {
	subcommands.Register(&ping)
}
