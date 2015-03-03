package create

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"

	clibazil "bazil.org/bazil/cli"
	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/control/wire"
	"github.com/golang/protobuf/proto"
)

type createCommand struct {
	subcommands.Description
	Arguments struct {
		VolumeName string
	}
}

func (cmd *createCommand) Run() error {
	req := wire.VolumeCreateRequest{
		VolumeName: cmd.Arguments.VolumeName,
	}
	buf, err := proto.Marshal(&req)
	if err != nil {
		return err
	}
	resp, err := clibazil.Bazil.Control.Post(
		"http+unix://bazil/control/volumeCreate",
		"binary/x.bazil.control.volumeCreateRequest",
		bytes.NewReader(buf),
	)
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

var create = createCommand{
	Description: "create a new volume",
}

func init() {
	subcommands.Register(&create)
}
