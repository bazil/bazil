package add

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"bazil.org/bazil/cas/chunks"
	clicas "bazil.org/bazil/cli/debug/cas"
	"bazil.org/bazil/cliutil/subcommands"
)

type addCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		Type  string
		Level uint8
	}
}

func (c *addCommand) Run() error {
	var buf bytes.Buffer
	const kB = 1024
	const MB = 1024 * kB
	const Max = 256 * MB
	n, err := io.CopyN(&buf, os.Stdin, Max)
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading standard input: %v", err)
	}
	if n == Max {
		return errors.New("aborting because chunk is unreasonably big")
	}

	chunk := &chunks.Chunk{
		Type:  c.Arguments.Type,
		Level: c.Arguments.Level,
		Buf:   buf.Bytes(),
	}
	key, err := clicas.CAS.State.Store.Add(chunk)
	if err != nil {
		return err
	}
	_, err = fmt.Printf("%v\n", key)
	if err != nil {
		return err
	}
	return nil
}

var add = addCommand{
	Description: "add a chunk to CAS",
	Synopsis:    "TYPE LEVEL <FILE",
}

func init() {
	subcommands.Register(&add)
}
