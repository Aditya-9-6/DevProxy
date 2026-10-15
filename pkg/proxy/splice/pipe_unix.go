//go:build !windows

package splice

import "syscall"

func createPipe() (int, int, error) {
	var fds [2]int
	if err := syscall.Pipe(fds[:]); err != nil {
		return 0, 0, err
	}
	return fds[0], fds[1], nil
}

func closePipe(r, w int) {
	_ = syscall.Close(r)
	_ = syscall.Close(w)
}
