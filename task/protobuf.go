// +build task

package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func gopath() []string {
	return filepath.SplitList(os.Getenv("GOPATH"))
}

func includeArgs(gopath []string) []string {
	l := make([]string, len(gopath))
	for i, p := range gopath {
		l[i] = "-I" + filepath.Join(p, "src")
	}
	return l
}

func sourceDir() (dir string, ok bool) {
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", ok
	}
	dir = filepath.Dir(file)

	// if the last segment is "task", strip that out; allows
	// segregating task files in a subdir
	parent, taskDir := filepath.Split(dir)
	if taskDir == "task" {
		dir = parent
	}

	return dir, ok
}

type WalkDirFunc func(path string, info os.FileInfo, children []os.FileInfo, err error) error

func walk(path string, info os.FileInfo, walkFn WalkDirFunc) error {
	children, err := ioutil.ReadDir(path)
	err = walkFn(path, info, children, err)
	if err != nil {
		return err
	}
	for _, fi := range children {
		// let walkFn set to nil any children it wants ignored
		if fi == nil {
			continue
		}
		if fi.IsDir() {
			err = walk(filepath.Join(path, fi.Name()), fi, walkFn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func WalkDirs(root string, walkFn WalkDirFunc) error {
	info, err := os.Lstat(root)
	if err != nil {
		return walkFn(root, nil, nil, err)
	}
	return walk(root, info, walkFn)
}

func main() {
	src, ok := sourceDir()
	if !ok {
		log.Fatal("cannot determine source directory")
	}
	err := WalkDirs(src, func(path string, info os.FileInfo, children []os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		var protos []string

		for i, fi := range children {
			if fi.Name()[0] == '.' {
				// skip hidden files and subdirs
				children[i] = nil
				continue
			}

			if fi.Mode().IsRegular() && filepath.Ext(fi.Name()) == ".proto" {
				// make a filename that looks like go import paths,
				// not absolute path, to keep protoc happy; it doesn't
				// understand multiple paths for same file

				// this should lead us to $P/src of the GOPATH entry that contains us
				goPathEntry := filepath.Join(src, "..", "..")
				p, err := filepath.Rel(goPathEntry, filepath.Join(path, fi.Name()))
				if err != nil {
					log.Fatalf("cannot make pretty path: %v", err)
				}
				protos = append(protos, p)
			}
		}

		if len(protos) == 0 {
			return nil
		}

		// protoc -I. -I$GOPATH/src --gogo_out=. foo.proto bar.proto
		var args []string
		args = append(args, "-I.")
		args = append(args, includeArgs(gopath())...)
		args = append(args, "--gogo_out=.")
		args = append(args, protos...)
		cmd := exec.Command("protoc", args...)
		// this should lead us to $P/src of the GOPATH entry that contains us
		//
		// relative entries in $GOPATH will break because of this
		// chdir -- but they're a bad idea anyway
		cmd.Dir = filepath.Join(src, "..", "..")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatalf("processing protobuf sources: %v", err)
	}
}
