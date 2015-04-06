package decode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"bazil.org/bazil/cliutil/subcommands"
	"bazil.org/bazil/fs/clock"
)

type decodeCommand struct {
	subcommands.Description
	subcommands.Synopsis
}

func (cmd *decodeCommand) Run() error {
	var buf bytes.Buffer
	const Max = 4096
	n, err := io.CopyN(&buf, os.Stdin, Max)
	if err != nil && err != io.EOF {
		return fmt.Errorf("reading standard input: %v", err)
	}
	if n == Max {
		return errors.New("aborting because clock is unreasonably big")
	}

	var c clock.Clock
	if err := c.UnmarshalBinary(buf.Bytes()); err != nil {
		return err
	}
	fmt.Printf("%v\n", c)
	return nil
}

var decode = decodeCommand{
	Description: "decode a logical clock",
	Synopsis:    "<FILE",
}

func init() {
	subcommands.Register(&decode)
}
