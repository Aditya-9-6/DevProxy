//go:build linux

package splice

import "syscall"

// Splice performs a zero-copy transfer from src to dst using the kernel pipe.
func Splice(src, dst int, pipe PipePair, len int) (int, error) {
	n, err := syscall.Splice(src, nil, pipe.Writer, nil, len, 0)
	if err != nil || n == 0 {
		return int(n), err
	}
	n2, err2 := syscall.Splice(pipe.Reader, nil, dst, nil, int(n), 0)
	return int(n2), err2
}
