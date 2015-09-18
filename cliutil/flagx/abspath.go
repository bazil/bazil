package flagx

import (
	"errors"
	"flag"
	"path/filepath"
)

// AbsPath returns a flag.Value that wraps the given string and sets
// it to an absolute path.
type AbsPath string

var _ flag.Value = (*AbsPath)(nil)

func (a AbsPath) String() string {
	return string(a)
}

var ErrEmptyPath = errors.New("empty path not allowed")

func (a *AbsPath) Set(value string) error {
	if value == "" {
		return ErrEmptyPath
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return err
	}
	*a = AbsPath(path)
	return nil
}
