package tcpee

import (
	"syscall"

	"codeberg.org/gruf/go-errors"
)

func SetSocketIPTransparent(network string, address string, c syscall.RawConn) error {
	var fnErr error
	err := c.Control(func(fd uintptr) {
		// Attempt to set IP_TRANSPARENT sock opt
		fnErr = syscall.SetsockoptInt(
			int(fd),
			syscall.SOL_IP,
			syscall.IP_TRANSPARENT,
			1,
		)
		if fnErr != nil {
			return
		}

		// Check that IP_TRANSPARENT was set
		var val int
		val, fnErr = syscall.GetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_TRANSPARENT)
		if fnErr != nil {
			return
		}
		if val != 1 {
			fnErr = errors.New("tcpee: failed to set IP_TRANSPARENT")
		}
	})
	if err != nil {
		return err
	}
	return fnErr
}
