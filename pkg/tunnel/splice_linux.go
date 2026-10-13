//go:build linux

package tunnel

import (
	"net"
	"syscall"
)

// SpliceCopy performs zero-copy data transfer between two TCP sockets using the splice syscall.
func SpliceCopy(dst, src *net.TCPConn) (written int64, err error) {
	srcFile, err := src.File()
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := dst.File()
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	pipe := make([]int, 2)
	if err := syscall.Pipe(pipe); err != nil {
		return 0, err
	}
	defer syscall.Close(pipe[0])
	defer syscall.Close(pipe[1])

	for {
		n, err := syscall.Splice(int(srcFile.Fd()), nil, pipe[1], nil, 65536, 0)
		if n == 0 {
			break
		}
		if err != nil {
			return written, err
		}

		_, err = syscall.Splice(pipe[0], nil, int(dstFile.Fd()), nil, int(n), 0)
		if err != nil {
			return written, err
		}
		written += int64(n)
	}
	return written, nil
}
