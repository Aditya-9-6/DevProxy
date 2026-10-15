//go:build windows

package splice

import "syscall"

func createPipe() (int, int, error) {
	var fds [2]syscall.Handle
	if err := syscall.Pipe(fds[:]); err != nil {
		return 0, 0, err
	}
	return int(fds[0]), int(fds[1]), nil
}

func closePipe(r, w int) {
	_ = syscall.Close(syscall.Handle(r))
	_ = syscall.Close(syscall.Handle(w))
}
