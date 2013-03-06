package env

import (
	"syscall"
)

func init() {
	MyUID = uint32(syscall.Getuid())
	MyGID = uint32(syscall.Getgid())
}
