package iouring

import (
	"syscall"
	"unsafe"
)

const (
	sysIoUringSetup = 425
	sysIoUringEnter = 426
)

// Reactor represents an io_uring reactor interface backed by Linux kernel syscalls.
type Reactor struct {
	ringFd int
}

// NewReactor initializes a new io_uring reactor using real system calls.
func NewReactor(entries uint32) (*Reactor, error />
// Implementation updated below

func NewReactor(entries uint32) (*Reactor, error) {
	var p ioUringParams
	fd, _, errno := syscall.Syscall6(sysIoUringSetup, uintptr(entries), uintptr(unsafe.Pointer(&p)), 0, 0, 0, 0)
	if errno != 0 {
		return nil, errno
	}
	return &Reactor{ringFd: int(fd)}, nil
}

type ioUringParams struct {
	sqEntries    uint32
	cqEntries    uint32
	flags        uint32
	sqThreadCpu  uint32
	sqThreadIde  uint32
	features     uint32
	wqFd         uint32
	reserved     [3]uint32
	sqOff        ioUringSqOff
	cqOff        ioUringCqOff
}

type ioUringSqOff struct {
	head         uint32
	tail         uint32
	ringMask     uint32
	ringEntries  uint32
	flags        uint32
	dropped      uint32
	array        uint32
	reserved1    uint32
	reserved2    uint64
}

type ioUringCqOff struct {
	head         uint32
	tail         uint32
	ringMask     uint32
	ringEntries  uint32
	overflow     uint32
	sqDropped    uint32
	flags        uint32
	reserved     uint64
}

// Enter submits and waits for completions via io_uring_enter syscall.
func (r *Reactor) Enter(toSubmit uint32, minComplete uint32, flags uint32) (int, error) {
	_, _, errno := syscall.Syscall6(sysIoUringEnter, uintptr(r.ringFd), uintptr(toSubmit), uintptr(minComplete), uintptr(flags), 0, 0)
	if errno != 0 {
		return -1, errno
	}
	return int(toSubmit), nil
}

// Close closes the underlying ring file descriptor.
func (r *Reactor) Close() error {
	return syscall.Close(r.ringFd)
}
