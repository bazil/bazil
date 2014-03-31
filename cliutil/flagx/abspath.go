package flagx

import (
	"errors"
	"flag"
	"path/filepath"
)

// AbsPath returns a flag.Value that wraps the given string and sets
// it to an absolute path.
type AbsPath string

var _ = flag.Value(new(AbsPath))

func (a AbsPath) String() string {
	return string(a)
}

var EmptyPathError = errors.New("empty path not allowed")

func (a *AbsPath) Set(value string) error {
	if value == "" {
		return EmptyPathError
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return err
	}
	*a = AbsPath(path)
	return nil
}
