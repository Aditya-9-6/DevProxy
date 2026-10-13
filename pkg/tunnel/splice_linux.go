//go:build linux

package tunnel

import (
	"net"
	"syscall"
)

const (
	// Ensure safe fallback if specific splice flags are missing in older target toolchains/kernels
	localSpliceFNonblock = 0x02
	localSpliceFMore     = 0x04
)

// SpliceCopy performs zero-copy data transfer between two TCP sockets using the splice syscall and SyscallConn.
func SpliceCopy(dst, src *net.TCPConn) (written int64, err error) {
	srcRaw, err := src.SyscallConn()
	if err != nil {
		return 0, err
	}

	dstRaw, err := dst.SyscallConn()
	if err != nil {
		return 0, err
	}

	pipe := make([]int, 2)
	if err := syscall.Pipe(pipe); err != nil {
		return 0, err
	}
	defer syscall.Close(pipe[0])
	defer syscall.Close(pipe[1])

	var srcFd, dstFd int
	err = srcRaw.Control(func(fd uintptr) {
		srcFd = int(fd)
	})
	if err != nil {
		return 0, err
	}

	err = dstRaw.Control(func(fd uintptr) {
		dstFd = int(fd)
	})
	if err != nil {
		return 0, err
	}

	flags := int(localSpliceFNonblock | localSpliceFMore)

	for {
		n, sErr := syscall.Splice(srcFd, nil, pipe[1], nil, 65536, flags)
		if n == 0 {
			break
		}
		if sErr != nil {
			if sErr == syscall.EAGAIN {
				continue
			}
			return written, sErr
		}

		_, sErr = syscall.Splice(pipe[0], nil, dstFd, nil, int(n), flags)
		if sErr != nil {
			if sErr == syscall.EAGAIN {
				continue
			}
			return written, sErr
		}
		written += int64(n)
	}

	return written, nil
}
