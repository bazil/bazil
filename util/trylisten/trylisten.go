package trylisten

import (
	"net"
	"os"
	"syscall"
)

// isInUse returns if the error was EADDRINUSE, as expected from from bind(2).
func isInUse(err error) bool {
	opErr, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	sysErr, ok := opErr.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	return sysErr.Err == syscall.EADDRINUSE
}

// ListenTCP tries to listen on the port requested, falling back to a
// dynamic port when needed.
func ListenTCP(network string, laddr *net.TCPAddr) (*net.TCPListener, error) {
	// try the address as requested
	l, err := net.ListenTCP(network, laddr)
	switch {
	case err == nil:
		// got it
		return l, nil
	case isInUse(err):
		// fall back to dynamic port allocation
		return net.ListenTCP(network, &net.TCPAddr{
			IP:   laddr.IP,
			Port: 0,
			Zone: laddr.Zone,
		})
	default:
		// no idea
		return nil, err
	}
}
