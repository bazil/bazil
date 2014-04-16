package hash

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"bazil.org/bazil/cas/chunks"
	"bazil.org/bazil/cas/chunks/chunkutil"
	"bazil.org/bazil/cliutil/subcommands"
)

type hashCommand struct {
	subcommands.Description
	subcommands.Synopsis
	Arguments struct {
		Type  string
		Level uint8
	}
}

func (c *hashCommand) Run() error {
	var buf bytes.Buffer
	const kB = 1024
	const MB = 1024
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
	key := chunkutil.Hash(chunk)
	fmt.Printf("%s\n", key)
	return nil
}

var hash = hashCommand{
	Description: "compute hash of a single chunk of data",
	Synopsis:    "TYPE LEVEL <FILE",
}

func init() {
	subcommands.Register(&hash)
}
