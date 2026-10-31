//go:build linux

package splice

import (
	"context"
	"io"
	"net"
	"syscall"
)

const spliceFNonblock = 0x2

// SpliceCopy copies data using the Linux splice(2) syscall when available,
// falling back gracefully if non-TCP or TLS-wrapped.
func SpliceCopy(ctx context.Context, dst net.Conn, src net.Conn) (int64, error) {
	sTCP, srcIsTCP := src.(*net.TCPConn)
	dTCP, dstIsTCP := dst.(*net.TCPConn)

	if !srcIsTCP || !dstIsTCP {
		return fallbackCopy(ctx, dst, src)
	}

	srcFile, err := sTCP.File()
	if err != nil {
		return fallbackCopy(ctx, dst, src)
	}
	defer srcFile.Close()

	dstFile, err := dTCP.File()
	if err != nil {
		return fallbackCopy(ctx, dst, src)
	}
	defer dstFile.Close()

	var pipeFds [2]int
	if err := syscall.Pipe2(pipeFds[:], syscall.O_NONBLOCK); err != nil {
		return fallbackCopy(ctx, dst, src)
	}
	defer syscall.Close(pipeFds[0])
	defer syscall.Close(pipeFds[1])

	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}

		n, err := syscall.Splice(int(srcFile.Fd()), nil, pipeFds[1], nil, 65536, spliceFNonblock)
		if n > 0 {
			_, wErr := syscall.Splice(pipeFds[0], nil, int(dstFile.Fd()), nil, int(n), spliceFNonblock)
			if wErr != nil && wErr != syscall.EAGAIN {
				return total, wErr
			}
			total += n
		}
		if err != nil {
			if err == syscall.EAGAIN {
				continue
			}
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}
		if n == 0 {
			break
		}
	}
	return total, nil
}

func fallbackCopy(ctx context.Context, dst net.Conn, src net.Conn) (int64, error) {
	buf := make([]byte, 32*1024)
	var total int64
	for {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		default:
		}
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				total += int64(nw)
			}
			if ew != nil {
				return total, ew
			}
		}
		if err != nil {
			if err == io.EOF {
				return total, nil
			}
			return total, err
		}
	}
}
