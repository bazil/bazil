package server

import (
	"errors"
	"os"
	"syscall"
)

func lock(lockPath string) (*os.File, error) {
	lockFile, err := os.Create(lockPath)
	if err != nil {
		return nil, err
	}

	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		if err == syscall.EWOULDBLOCK {
			return nil, errors.New("another server is already running")
		}
		return nil, err
	}
	// closing lockFile will release the lock
	return lockFile, nil
}
