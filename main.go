package main

import (
	"os"

	"bazil.org/bazil/cli"
)

func main() {
	code := cli.Main()
	os.Exit(code)
}
