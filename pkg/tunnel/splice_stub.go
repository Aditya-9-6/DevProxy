//go:build !linux

package tunnel

import "net"

// SpliceCopy is a stub for non-Linux platforms, always returning 0 to trigger io.CopyBuffer fallback.
func SpliceCopy(dst, src *net.TCPConn) (int64, error) {
	return 0, nil
}
