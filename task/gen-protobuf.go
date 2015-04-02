// +build task

package main

import (
	"errors"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func gopath() []string {
	return filepath.SplitList(os.Getenv("GOPATH"))
}

func includeArgs(gopath []string) []string {
	l := make([]string, 0, len(gopath))
	for _, p := range gopath {
		l = append(l, "-I"+filepath.Join(p, "src"))
	}
	return l
}

func srcRoot(abspath string) (string, error) {
	pkg, err := build.ImportDir(abspath, build.FindOnly)
	if err != nil {
		return "", err
	}
	if pkg.SrcRoot == "" {
		return "", errors.New("cannot determine good GOPATH/src")
	}
	return pkg.SrcRoot, nil
}

func listProtos() ([]string, error) {
	var protos []string
	children, err := ioutil.ReadDir(".")
	if err != nil {
		return nil, err
	}
	for _, fi := range children {
		if fi.Name()[0] == '.' {
			// skip hidden files and subdirs
			continue
		}
		if filepath.Ext(fi.Name()) != ".proto" {
			continue
		}
		if !fi.Mode().IsRegular() {
			continue
		}
		protos = append(protos, fi.Name())
	}
	return protos, nil
}

func process() error {
	protos, err := listProtos()
	if err != nil {
		return err
	}
	if len(protos) == 0 {
		return errors.New("no proto files found")
	}

	cwd, err := filepath.Abs(".")
	if err != nil {
		return err
	}

	src, err := srcRoot(cwd)
	if err != nil {
		return err
	}

	// protoc -I$GOPATH/src --go_out=plugins=grpc:. $GOPATH/src/bazil.org/bazil/quux/foo.proto $GOPATH/src/bazil.org/bazil/quux/bar.proto
	var args []string
	args = append(args, includeArgs(gopath())...)
	args = append(args, "--go_out=plugins=grpc:.")
	for _, proto := range protos {
		// keep paths absolute to make protoc happy; it doesn't
		// understand multiple paths for same file
		args = append(args, filepath.Join(cwd, proto))
	}

	cmd := exec.Command("protoc", args...)
	// this should lead us to $P/src of the GOPATH entry that contains us
	//
	// relative entries in $GOPATH will break because of this
	// chdir -- but they're a bad idea anyway
	cmd.Dir = src
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("processing protobuf sources: %v", err)
	}
	return nil
}

var prog = filepath.Base(os.Args[0])

func main() {
	log.SetFlags(0)
	log.SetPrefix(prog + ": ")

	flag.Parse()
	if flag.NArg() != 0 {
		flag.Usage()
		os.Exit(2)
	}

	if err := process(); err != nil {
		log.Fatal(err)
	}
}
