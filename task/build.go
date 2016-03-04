// +build task

package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

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

func goBuild(src string, action string, args ...string) error {
	cmd := exec.Command(
		"go", action,
		"-v",
	)
	cmd.Args = append(cmd.Args, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func gitVersion() (string, error) {
	cmd := exec.Command(
		"git", "describe",
		"--match", "release/*",
		"--dirty=-edited",
	)
	cmd.Stderr = os.Stderr
	buf, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(buf), nil
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("# ")
	src, ok := sourceDir()
	if !ok {
		log.Fatal("cannot determine source directory")
	}
	if err := os.Chdir(src); err != nil {
		log.Fatalf("cannot change to source directory: %v", err)
	}

	log.Print("build bazil")
	version, err := gitVersion()
	if err != nil {
		log.Fatalf("git describe: %v", err)
	}

	if err := goBuild(src, "build", "-v",
		"-ldflags", "-X bazil.org/bazil/version.Version="+version,
		"bazil.org/bazil",
	); err != nil {
		log.Fatalf("go build of bazil: %v", err)
	}
}
