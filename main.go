package main

import (
	"os"

	"bazil.org/bazil/cli"
)

//go:generate go run task/gen-imports.go -o commands.gen.go bazil.org/bazil/cli/...

func main() {
	code := cli.Main()
	os.Exit(code)
}
